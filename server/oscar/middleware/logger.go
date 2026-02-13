package middleware

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/pchchv/go-icq/wire"
)

const LevelTrace = slog.Level(-8)

type RouteLogger struct {
	Logger *slog.Logger
}

func (rt RouteLogger) LogRequest(ctx context.Context, inFrame wire.SNACFrame, inSNAC any) {
	LogRequest(ctx, rt.Logger, inFrame, inSNAC)
}

func (rt RouteLogger) LogRequestError(ctx context.Context, inFrame wire.SNACFrame, err error) {
	LogRequestError(ctx, rt.Logger, inFrame, err)
}

func (rt RouteLogger) LogRequestAndResponse(ctx context.Context, inFrame wire.SNACFrame, inSNAC any, outFrame wire.SNACFrame, outSNAC any) {
	msg := "client request -> server response"
	switch {
	case rt.Logger.Enabled(ctx, LevelTrace):
		rt.Logger.LogAttrs(
			ctx,
			LevelTrace,
			msg,
			snacLogGroupWithPayload("request", inFrame, inSNAC),
			snacLogGroupWithPayload("response", outFrame, outSNAC),
		)
	case rt.Logger.Enabled(ctx, slog.LevelDebug):
		rt.Logger.LogAttrs(ctx, slog.LevelDebug, msg, snacLogGroup("request", inFrame), snacLogGroup("response", outFrame))
	}
}

type handler struct {
	slog.Handler
}

func (h handler) Handle(ctx context.Context, r slog.Record) error {
	if sn := ctx.Value("screenName"); sn != nil {
		r.AddAttrs(slog.Attr{Key: "screenName", Value: slog.StringValue(sn.(fmt.Stringer).String())})
	}

	if ip := ctx.Value("ip"); ip != nil {
		r.AddAttrs(slog.Attr{Key: "ip", Value: slog.StringValue(ip.(string))})
	}

	return h.Handler.Handle(ctx, r)
}

func (h handler) WithGroup(name string) slog.Handler {
	return h.Handler.WithGroup(name)
}

func (h handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return handler{h.Handler.WithAttrs(attrs)}
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
