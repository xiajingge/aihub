package channel

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/privacy"
	"github.com/gaohao-creator/executors"
	"github.com/xiajignge/aihub/internal/domain"
	"github.com/xiajignge/aihub/internal/domain/llm"
	"github.com/xiajignge/aihub/internal/domain/transformer/openai"
	"github.com/xiajignge/aihub/internal/ent"
	"github.com/xiajignge/aihub/internal/ent/channel"
	"github.com/xiajingge/logger"
	"go.uber.org/fx"
)

type ChannelService struct {
	Channels  []*llm.Channel
	Executors executors.ScheduledExecutor
	Ent       *ent.Client
	// latestUpdate 记录最新的 channel 更新时间，用于优化定时加载
	latestUpdate time.Time
	logger       logger.LoggerV1
}

// loadChannelsPeriodic 定时任务入口，加载通道并记录日志。
func (svc *ChannelService) loadChannelsPeriodic(ctx context.Context) {
	err := svc.loadChannels(ctx)
	if err != nil {
		// 记录加载通道失败的错误日志
		svc.logger.Error("failed to load channels", logger.Error(err))
	}
}

// loadChannels 从数据库加载启用的通道并构建出站转换器缓存。
func (svc *ChannelService) loadChannels(ctx context.Context) error {
	// 绕过权限检查，内部加载通道配置
	ctx = privacy.DecisionContext(ctx, privacy.Allow)

	// 检查是否有 channels 被修改
	// 查询最新更新时间的通道
	latestUpdatedChannel, err := svc.Ent.Channel.Query().
		Order(ent.Desc(channel.FieldUpdatedAt)).
		First(ctx)
	// 忽略未找到通道的情况
	if err != nil && !ent.IsNotFound(err) {
		return err
	}

	// 如果没有找到任何 channels，latestUpdate 会是 nil
	if latestUpdatedChannel != nil {
		// 如果最新的更新时间早于或等于我们记录的时间，说明没有新的修改
		if !latestUpdatedChannel.UpdatedAt.After(svc.latestUpdate) {
			svc.logger.Debug("no new channels updated")
			return nil
		}
		// 更新最新的修改时间记录
		svc.latestUpdate = latestUpdatedChannel.UpdatedAt
	} else {
		// 如果没有 channels，确保 latestUpdate 是零值时间
		svc.latestUpdate = time.Time{}
	}

	// 查询启用的通道并按权重排序
	entities, err := svc.Ent.Channel.Query().
		Where(channel.StatusEQ(channel.StatusEnabled)).
		Order(ent.Desc(channel.FieldOrderingWeight)).
		All(ctx)
	if err != nil {
		return err
	}

	var channels []*llm.Channel

	for _, c := range entities {
		// 构建通道与出站转换器
		channel, err := svc.buildChannel(c)
		if err != nil {
			// 记录单个通道构建失败，继续处理其他通道
			svc.logger.Warn("failed to build channel",
				logger.String("channel", c.Name),
				logger.String("type", c.Type.String()),
				logger.Error(err),
			)

			continue
		}

		// 记录出站转换器创建成功
		svc.logger.Debug("created outbound transformer", logger.String("channel", c.Name), logger.String("type", c.Type.String()))

		channels = append(channels, channel)
	}

	// 更新内存中的通道缓存
	svc.Channels = channels

	return nil
}

// buildChannel 根据通道类型构建对应的出站转换器。
func (svc *ChannelService) buildChannel(c *ent.Channel) (*llm.Channel, error) {
	// TODO SUPPORT more providers.
	switch c.Type {
	case channel.TypeOpenai, channel.TypeDeepseek, channel.TypeDoubao, channel.TypeMoonshot:
		// 构建 OpenAI 兼容出站转换器
		transformer, err := openai.NewOutboundTransformer(c.BaseURL, c.Credentials.APIKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create outbound transformer: %w", err)
		}

		return &llm.Channel{
			Channel:  c,
			Outbound: transformer,
		}, nil
	// TODO  openai.NewFakeTransformer()
	//case channel.TypeOpenaiFake:
	//	// 使用 OpenAI 假实现用于测试
	//
	//	fakeTransformer := openai.NewFakeTransformer()
	//	return &llm.Channel{
	//		Channel:  c,
	//		Outbound: fakeTransformer,
	//	}, nil
	default:
		return nil, errors.New("unknown channel type")
	}
}

// ChooseChannels 根据请求模型筛选可用通道列表。
func (svc *ChannelService) ChooseChannels(
	ctx context.Context,
	chatReq *domain.Request,
) ([]*llm.Channel, error) {
	var channels []*llm.Channel

	for _, channel := range svc.Channels {
		// 仅保留支持目标模型的通道
		if channel.IsModelSupported(chatReq.Model) {
			channels = append(channels, channel)
		}
	}

	return channels, nil
}

func (svc *ChannelService) GetChannelForTest(ctx context.Context, channelID int) (*llm.Channel, error) {
	// 绕过权限检查，允许读取禁用通道
	ctx = privacy.DecisionContext(ctx, privacy.Allow)

	// Get the channel entity from database (including disabled ones)
	// 读取通道实体（包含禁用通道）
	entity, err := svc.Ent.Channel.Get(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("channel not found: %w", err)
	}

	// 复用通道构建逻辑
	return svc.buildChannel(entity)
}

type ChannelServiceParams struct {
	fx.In

	Executor executors.ScheduledExecutor
	Client   *ent.Client
	Logger   logger.LoggerV1
}

func NewChannelService(params ChannelServiceParams) *ChannelService {
	svc := &ChannelService{
		Executors: params.Executor,
		Ent:       params.Client,
		logger:    params.Logger,
	}

	// 初始化加载通道缓存
	err := svc.loadChannels(context.Background())
	if err != nil {
		panic(err)
	}

	// 使用定时任务周期性刷新通道缓存
	_, err = params.Executor.ScheduleFuncAtCronRate(
		svc.loadChannelsPeriodic,
		executors.CRONRule{Expr: "*/1 * * * *"},
	)
	if err != nil {
		panic(err)
	}

	return svc
}
