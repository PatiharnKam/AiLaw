package messageshistory

import "context"

type Service struct {
	storage MessageStorage
}

func NewService(storage MessageStorage) *Service {
	return &Service{
		storage: storage,
	}
}

func (s *Service) GetMessageHistoryService(ctx context.Context, req MessageHistoryRequest) ([]MessageHistoryResponse, error) {
	resp, err := s.storage.GetMessageHistoryStorage(ctx, req.SessionId)
	if err != nil {
		return nil, err
	}

	messageResp := []MessageHistoryResponse{}
	for _, data := range resp {
		messageResp = append(messageResp, MessageHistoryResponse{
			SessionId: data.SessionId,
			MessageId: data.MessageId,
			Role:      data.Role,
			Content:   data.Content,
			CreatedAt: data.CreatedAt,
			Feedback:  data.Feedback,
		})
	}

	return messageResp, nil
}
