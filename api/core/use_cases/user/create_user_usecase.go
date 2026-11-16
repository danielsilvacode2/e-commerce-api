package use_cases

import (
	"e-commerce/core/apperrors"
	"e-commerce/core/entities/user"
	"e-commerce/core/ports"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
)

type CreateUserUseCase struct {
	repository   entities.UserRepository
	hashService  ports.HashService
	emailService ports.EmailService
	eventQueue   ports.EventUserQueue
}

func NewCreateUserUseCase(

	repository entities.UserRepository,
	passwordService ports.HashService,
	emailService ports.EmailService,
	eventQueue ports.EventUserQueue,

) *CreateUserUseCase {
	return &CreateUserUseCase{
		repository:   repository,
		hashService:  passwordService,
		emailService: emailService,
		eventQueue:   eventQueue,
	}
}

func (useCase CreateUserUseCase) UseCase(user entities.User) (*entities.User, error) {

	user.ID = uuid.New()
	passwordHash, err := useCase.hashService.HashGenerate(user.Password)

	if err != nil {
		return nil, err
	}

	user.Password = passwordHash

	if user.Role == "" {
		user.Role = "user"
	}

	user.NormalizeData()

	code := rand.Int64N(900000)
	codeFortm := fmt.Sprintf("%06d", code)

	codeHash, err := useCase.hashService.HashGenerate(codeFortm)

	if err != nil {
		return nil, err
	}

	user.VerificationCode = codeHash

	timeExpiresCode := time.Now().Add(15 * time.Minute)

	user.CodeExpires = timeExpiresCode

	err = useCase.repository.CreateUser(user)
	if err != nil {

		if errors.Is(err, apperrors.ErrDatabaseUnavailable) {

			user.VerificationCode = codeFortm
			if err = useCase.eventQueue.PublishEvent(ports.QueueCreateUser, user); err != nil {
				return nil, err
			}

			return &user, nil
		}

		if errors.Is(err, apperrors.ErrConflict) {
			go func() {
				if err := useCase.emailService.SendSuspiciousLogin(user.Email); err != nil {
					log.Printf("[ERROR] [SuspiciousLoginEmail] Falha ao enviar para userID=%s email=%s: %v", user.ID, user.Email, err)
				} else {
					log.Printf("[INFO] [SuspiciousLoginEmail] E-mail enviado com sucesso para userID=%s email=%s", user.ID, user.Email)
				}
			}()

			return nil, err
		}

		return nil, err
	}

	go func() {

		if err := useCase.emailService.SendVerificationCode(user.Email, codeFortm, "Verificar email"); err != nil {
			log.Printf("[ERROR] [VerificationCodeEmail] Falha ao enviar para userID=%s email=%s: %v", user.ID, user.Email, err)
		} else {
			log.Printf("[INFO] [VerificationCodeEmail] E-mail enviado com sucesso para userID=%s email=%s", user.ID, user.Email)
		}

	}()

	return &user, nil
}
