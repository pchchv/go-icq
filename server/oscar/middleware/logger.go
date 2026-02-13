package middleware

import (
	"log/slog"

	"github.com/pchchv/go-icq/wire"
)

type RouteLogger struct {
	Logger *slog.Logger
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
