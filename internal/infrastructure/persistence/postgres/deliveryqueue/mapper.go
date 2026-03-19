package deliveryqueue

import "github.com/HiroLiang/tentserv-chat-server/internal/domain/deliveryqueue"

func toDomain(rec *DeliveryQueueRecord) *deliveryqueue.DeliveryQueue {
	return &deliveryqueue.DeliveryQueue{
		ID:          rec.ID,
		UserID:      rec.UserID,
		PayloadType: rec.PayloadType,
		Payload:     rec.Payload,
		Status:      rec.Status,
		CreatedAt:   rec.CreatedAt,
		DeliveredAt: rec.DeliveredAt,
	}
}

func toRecord(item *deliveryqueue.DeliveryQueue) *DeliveryQueueRecord {
	return &DeliveryQueueRecord{
		ID:          item.ID,
		UserID:      item.UserID,
		PayloadType: item.PayloadType,
		Payload:     item.Payload,
		Status:      item.Status,
		CreatedAt:   item.CreatedAt,
		DeliveredAt: item.DeliveredAt,
	}
}
