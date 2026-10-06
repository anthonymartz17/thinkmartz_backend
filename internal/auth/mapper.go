package auth

// toRegisterInput maps a RegisterRequest to the service-layer RegisterInput.
func toRegisterInput(req RegisterRequest) RegisterInput {
	return RegisterInput(req)
}

// toLoginInput maps a LoginRequest to the service-layer LoginInput.
func toLoginInput(req LoginRequest) LoginInput {
	return LoginInput(req)
}

// toUserResponse maps a domain User to its client-safe projection.
func toUserResponse(u User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Username:  u.Username,
		CreatedAt: u.CreatedAt,
	}
}
