package feedback

import "context"

type FeedbackService interface {
	FeedbackService(ctx context.Context, req FeedbackRequest) error
}

type FeedbackStorage interface {
	FeedbackStorage(ctx context.Context, req FeedbackRequest) error
}

type FeedbackRequest struct {
	MessageID      string  `json:"messageID" validate:"required,uuid4"`
	Feedback       *int    `json:"feedback" validate:"omitempty,oneof=1 -1"`
	FeedbackDetail *string `json:"feedbackDetail"`
}
