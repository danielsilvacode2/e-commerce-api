package ports

import entities "e-commerce/core/entities/user"

type EventUserQueue interface {
	DeclareQueue(eventName string) error
	PublishEvent(eventName string, user entities.User) error
}

type EventeUserQueueConsumerDeclare interface{
	Consumer() error
	DeclareQueue(eventName string) error
}