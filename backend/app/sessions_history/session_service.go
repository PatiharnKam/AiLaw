package sessionshistory

import "context"

type Service struct {
	storage SessionStorage
}

func NewService(storage SessionStorage) *Service {
	return &Service{
		storage: storage,
	}
}

func (s *Service) GetSessionsHistoryService(ctx context.Context, req SessionsHistoryRequest) ([]SessionsHistoryResponse, error) {
	resp, err := s.storage.GetSessionsHistoryStorage(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	messageResp := []SessionsHistoryResponse{}
	for _, data := range resp {
		messageResp = append(messageResp, SessionsHistoryResponse{
			UserId:        data.UserId,
			SessionId:     data.SessionId,
			Title:         data.Title,
			CreatedAt:     data.CreatedAt,
			LastMessageAt: data.LastMessageAt,
		})
	}

	return messageResp, nil
}
