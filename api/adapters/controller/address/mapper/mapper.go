package mapper

import (
	"e-commerce/adapters/controller/address/dto"
	entitiesAddress "e-commerce/core/entities/address"
	"e-commerce/core/ports"
)

func DtoCreateRequestToAddress(req dto.CreateAddressRequestDTO) entitiesAddress.Address {
	return entitiesAddress.Address{
		Cep:          req.Cep,
		Street:       req.Street,
		Number:       req.Number,
		Complement:   req.Complement,
		Neighborhood: req.Neighborhood,
		City:         req.City,
		State:        req.State,
	}
}

func CepLookupResultToDto(res *ports.CepLookupResult) dto.AddressLookupResponseDTO {
	if res == nil {
		return dto.AddressLookupResponseDTO{}
	}
	return dto.AddressLookupResponseDTO{
		Cep:          res.Cep,
		Street:       res.Street,
		Neighborhood: res.Neighborhood,
		City:         res.City,
		State:        res.State,
	}
}

func SavedAddressToDto(addr entitiesAddress.Address) dto.SavedAddressResponseDTO {
	return SavedAddressToDtoWithWarning(addr, "")
}

func SavedAddressToDtoWithWarning(addr entitiesAddress.Address, warning string) dto.SavedAddressResponseDTO {
	return dto.SavedAddressResponseDTO{
		Id:                       addr.Id,
		Cep:                      addr.Cep,
		Street:                   addr.Street,
		Number:                   addr.Number,
		Complement:               addr.Complement,
		Neighborhood:             addr.Neighborhood,
		City:                     addr.City,
		State:                    addr.State,
		AddressValidationWarning: warning,
	}
}

func SavedAddressesToDto(list []entitiesAddress.Address) []dto.SavedAddressResponseDTO {
	res := make([]dto.SavedAddressResponseDTO, 0, len(list))
	for _, addr := range list {
		res = append(res, SavedAddressToDto(addr))
	}
	return res
}
