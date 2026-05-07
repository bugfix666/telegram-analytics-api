package telegram

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/dcs"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"
	"golang.org/x/net/proxy"

	"telegram-analytics-api/internal/domain"
)

type Client struct {
	api      *telegram.Client
	raw      *tg.Client
	log      *zap.Logger
	ready    chan struct{}
	botToken string
}

func NewClient(apiID int, apiHash, sessionFile, botToken, proxyAddr string, log *zap.Logger) *Client {
	opts := telegram.Options{
		SessionStorage: &telegram.FileSessionStorage{Path: sessionFile},
		Logger:         log.Named("tg"),
	}

	if proxyAddr != "" {
		proxyURL, err := url.Parse(proxyAddr)
		if err != nil {
			log.Fatal("Failed to parse proxy URL", zap.String("proxy", proxyAddr), zap.Error(err))
		}
		dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
		if err != nil {
			log.Fatal("Failed to create proxy dialer", zap.String("proxy", proxyAddr), zap.Error(err))
		}
		contextDialer, ok := dialer.(proxy.ContextDialer)
		if !ok {
			log.Fatal("Proxy dialer does not support ContextDialer", zap.String("proxy", proxyAddr))
		}
		resolver := dcs.Plain(dcs.PlainOptions{
			Dial: contextDialer.DialContext,
		})
		opts.Resolver = resolver
		log.Info("Using proxy", zap.String("proxy", proxyAddr))
	}

	api := telegram.NewClient(apiID, apiHash, opts)
	return &Client{
		api:      api,
		log:      log,
		ready:    make(chan struct{}),
		botToken: botToken,
	}
}

func (c *Client) Start(ctx context.Context) error {
	return c.api.Run(ctx, func(ctx context.Context) error {
		if c.botToken != "" {
			c.log.Info("Authenticating as bot")
			if _, err := c.api.Auth().Bot(ctx, c.botToken); err != nil {
				return fmt.Errorf("bot auth: %w", err)
			}
		} else {
			c.log.Info("Starting interactive user authentication")
			flow := auth.NewFlow(&interactiveAuthenticator{}, auth.SendCodeOptions{})
			if err := c.api.Auth().IfNecessary(ctx, flow); err != nil {
				return fmt.Errorf("user auth: %w", err)
			}
		}
		c.raw = c.api.API()
		c.log.Info("Telegram client authenticated")
		close(c.ready)
		<-ctx.Done()
		return ctx.Err()
	})
}

func (c *Client) WaitReady() <-chan struct{} {
	return c.ready
}

func (c *Client) Stop(ctx context.Context) error {
	return nil
}

func (c *Client) SendMessage(ctx context.Context, chatID, text string) (int, error) {
	peer, err := c.resolvePeer(ctx, chatID)
	if err != nil {
		return 0, err
	}
	updates, err := message.NewSender(c.raw).To(peer).Text(ctx, text)
	if err != nil {
		return 0, err
	}
	return extractMessageID(updates)
}

func (c *Client) GetChatInfo(ctx context.Context, chatID string) (domain.ChatInfo, error) {
	peer, err := c.resolvePeer(ctx, chatID)
	if err != nil {
		return domain.ChatInfo{}, err
	}
	inputChan, ok := peer.(*tg.InputPeerChannel)
	if !ok {
		return domain.ChatInfo{}, fmt.Errorf("only channels/supergroups supported")
	}
	full, err := c.raw.ChannelsGetFullChannel(ctx, &tg.InputChannel{
		ChannelID:  inputChan.ChannelID,
		AccessHash: inputChan.AccessHash,
	})
	if err != nil {
		return domain.ChatInfo{}, err
	}
	var subs int
	var desc string
	switch ch := full.FullChat.(type) {
	case *tg.ChannelFull:
		subs = ch.ParticipantsCount
		desc = ch.About
	case *tg.ChatFull:
		if part, ok := ch.Participants.(*tg.ChatParticipants); ok {
			subs = len(part.Participants)
		}
		desc = ch.About
	default:
		return domain.ChatInfo{}, fmt.Errorf("unknown chat type")
	}
	return domain.ChatInfo{
		ID:          chatID,
		Subscribers: subs,
		Description: desc,
	}, nil
}

func (c *Client) IterateMessages(ctx context.Context, chatID string, since time.Time, limit int) ([]domain.Message, error) {
	peer, err := c.resolvePeer(ctx, chatID)
	if err != nil {
		return nil, err
	}
	inputChan, ok := peer.(*tg.InputPeerChannel)
	if !ok {
		return nil, fmt.Errorf("only channels/supergroups supported")
	}
	inputPeer := &tg.InputPeerChannel{
		ChannelID:  inputChan.ChannelID,
		AccessHash: inputChan.AccessHash,
	}
	var result []domain.Message
	offsetID := 0
	collected := 0
	for {
		msgs, err := c.raw.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
			Peer:     inputPeer,
			OffsetID: offsetID,
			Limit:    100,
		})
		if err != nil {
			return nil, err
		}
		batch := extractMessages(msgs)
		if len(batch) == 0 {
			break
		}
		for _, msg := range batch {
			msgTime := time.Unix(int64(msg.Date), 0)
			if !since.IsZero() && msgTime.Before(since) {
				return result, nil
			}
			reactions := make(map[string]int)
			if msg.Reactions.Results != nil {
				for _, r := range msg.Reactions.Results {
					if em, ok := r.Reaction.(*tg.ReactionEmoji); ok {
						reactions[em.Emoticon] = r.Count
					}
				}
			}
			result = append(result, domain.Message{
				ID:        msg.ID,
				Date:      msgTime,
				Views:     msg.Views,
				Forwards:  msg.Forwards,
				Reactions: reactions,
				Text:      msg.Message,
			})
			collected++
			if limit > 0 && collected >= limit {
				return result, nil
			}
			offsetID = msg.ID
		}
		if len(batch) < 100 {
			break
		}
	}
	return result, nil
}

func (c *Client) resolvePeer(ctx context.Context, chatID string) (tg.InputPeerClass, error) {
	if chatID == "" {
		return nil, fmt.Errorf("empty chat ID")
	}
	username := strings.TrimPrefix(chatID, "@")
	if username == "" {
		return nil, fmt.Errorf("empty username")
	}
	req := &tg.ContactsResolveUsernameRequest{Username: username}
	resolved, err := c.raw.ContactsResolveUsername(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(resolved.Chats) == 0 {
		return nil, fmt.Errorf("chat not found")
	}
	ch, ok := resolved.Chats[0].(*tg.Channel)
	if !ok {
		return nil, fmt.Errorf("not a channel or supergroup")
	}
	return &tg.InputPeerChannel{
		ChannelID:  ch.ID,
		AccessHash: ch.AccessHash,
	}, nil
}

func extractMessageID(updates tg.UpdatesClass) (int, error) {
	switch u := updates.(type) {
	case *tg.Updates:
		for _, update := range u.Updates {
			if upd, ok := update.(*tg.UpdateNewMessage); ok {
				if msg, ok := upd.Message.(*tg.Message); ok {
					return msg.ID, nil
				}
			}
		}
	case *tg.UpdateShortSentMessage:
		return u.ID, nil
	}
	return 0, fmt.Errorf("cannot extract message ID")
}

func extractMessages(messages tg.MessagesMessagesClass) []*tg.Message {
	var out []*tg.Message
	switch m := messages.(type) {
	case *tg.MessagesChannelMessages:
		for _, msg := range m.Messages {
			if msg, ok := msg.(*tg.Message); ok {
				out = append(out, msg)
			}
		}
	case *tg.MessagesMessages:
		for _, msg := range m.Messages {
			if msg, ok := msg.(*tg.Message); ok {
				out = append(out, msg)
			}
		}
	}
	return out
}
