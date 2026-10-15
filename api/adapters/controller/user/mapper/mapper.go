package mapper

import (
	"e-commerce/adapters/controller/user/dto"
	entities "e-commerce/core/entities/user"
)

func UserToResponseDTO(user entities.User) dto.UserResponseDTO{

	var userDto dto.UserResponseDTO

	userDto.Id = user.ID
	userDto.Name = user.Name
	userDto.Email = user.Email
	userDto.Role = user.Role
	userDto.Update_at = user.UpdatedAt
	userDto.Created_at = user.CreatedAt

	return userDto
}

func DtoCreateRequestToUser( dto dto.UserCreateRequestDTO) entities.User{

	var user entities.User

	user.Name = dto.Name
	user.Email = dto.Email
	user.Password = dto.Password

	return user
}


func DtoUpdateRequestToUser( dto dto.UserUpdateRequestDTO) entities.User{

	var user entities.User

	user.Name = dto.Name
	user.Email = dto.Email
	

	return user
}