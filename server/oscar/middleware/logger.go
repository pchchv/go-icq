package middleware

import (
	"context"
	"log/slog"

	"github.com/pchchv/go-icq/wire"
)

const LevelTrace = slog.Level(-8)

type RouteLogger struct {
	Logger *slog.Logger
}

func LogRequest(ctx context.Context, logger *slog.Logger, inFrame wire.SNACFrame, inSNAC any) {
	const msg = "client request"
	switch {
	case logger.Enabled(ctx, LevelTrace):
		logger.LogAttrs(ctx, LevelTrace, msg, snacLogGroupWithPayload("request", inFrame, inSNAC))
	case logger.Enabled(ctx, slog.LevelDebug):
		logger.LogAttrs(ctx, slog.LevelDebug, msg, snacLogGroup("request", inFrame))
	}
}

func LogRequestError(ctx context.Context, logger *slog.Logger, inFrame wire.SNACFrame, err error) {
	logger.LogAttrs(ctx, slog.LevelError, "client request error",
		slog.Group("request",
			slog.String("food_group", wire.FoodGroupName(inFrame.FoodGroup)),
			slog.String("sub_group", wire.SubGroupName(inFrame.FoodGroup, inFrame.SubGroup)),
		),
		slog.String("err", err.Error()),
	)
}

func snacLogGroupWithPayload(key string, outFrame wire.SNACFrame, outSNAC any) slog.Attr {
	return slog.Group(key,
		slog.String("food_group", wire.FoodGroupName(outFrame.FoodGroup)),
		slog.String("sub_group", wire.SubGroupName(outFrame.FoodGroup, outFrame.SubGroup)),
		slog.Any("snac_frame", outFrame),
		slog.Any("snac_payload", outSNAC),
	)
}

func snacLogGroup(key string, outFrame wire.SNACFrame) slog.Attr {
	return slog.Group(key,
		slog.String("food_group", wire.FoodGroupName(outFrame.FoodGroup)),
		slog.String("sub_group", wire.SubGroupName(outFrame.FoodGroup, outFrame.SubGroup)),
	)
}
