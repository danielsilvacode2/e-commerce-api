package mapper

import (
	"e-commerce/adapters/rabbitmq/dto"
	entities "e-commerce/core/entities/user"

	"github.com/google/uuid"
)

func UserToRabbitDTO(user entities.User) dto.RabbitMQUserDto {

	var dto dto.RabbitMQUserDto

	dto.ID = user.ID.String()
	dto.Name = user.Name
	dto.Email = user.Email
	dto.Password = user.Password
	dto.Role = user.Role
	dto.VerificationCode = user.VerificationCode
	dto.CodeExpires = user.CodeExpires
	dto.IsVerified = user.IsVerified
	dto.CreatedAt = user.CreatedAt
	dto.UpdatedAt = user.UpdatedAt

	return dto
}

func DtoToRabbitUser(dto dto.RabbitMQUserDto) (*entities.User, error) {

	var user entities.User

	userId, err := uuid.Parse(dto.ID)

	if err != nil {
		return nil, err
	}

	user.ID = userId
	user.Name = dto.Name
	user.Email = dto.Email
	user.Password = dto.Password
	user.Role = dto.Role
	user.CodeExpires = dto.CodeExpires
	user.VerificationCode = dto.VerificationCode
	user.IsVerified = dto.IsVerified
	user.CreatedAt = dto.CreatedAt
	user.UpdatedAt = dto.UpdatedAt

	return &user, nil
}
