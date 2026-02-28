package foodgroup

import (
	"context"

	"github.com/pchchv/go-icq/wire"
)

type StatsService struct{}

func NewStatsService() StatsService {
	return StatsService{}
}

// ReportEvents handles incoming stats events by acknowledging them without processing.
// This is a no-op implementation to satisfy the client's expectation of a response.
func (s StatsService) ReportEvents(ctx context.Context, inFrame wire.SNACFrame, _ wire.SNAC_0x0B_0x03_StatsReportEvents) wire.SNACMessage {
	return wire.SNACMessage{
		Frame: wire.SNACFrame{
			FoodGroup: wire.Stats,
			SubGroup:  wire.StatsReportAck,
			RequestID: inFrame.RequestID,
		},
		Body: wire.SNAC_0x0B_0x04_StatsReportAck{},
	}
}
