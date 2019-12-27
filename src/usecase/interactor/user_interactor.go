package interactor

import (
	"context"
	"time"

	"github.com/ezio1119/fishapp-auth/domain"
	"github.com/ezio1119/fishapp-auth/usecase/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserInteractor struct {
	UserRepository      repository.UserRepository
	BlackListRepository repository.BlackListRepository
	TokenInteractor     UTokenInteractor
	ContextTimeout      time.Duration
}

type UUserInteractor interface {
	GetByID(ctx context.Context, id int64) (*domain.User, error)
	Update(ctx context.Context, u *domain.User) error
	Create(ctx context.Context, u *domain.User) (*domain.TokenPair, error)
	Delete(ctx context.Context, id int64) error
	Login(ctx context.Context, email string, pass string) (*domain.User, *domain.TokenPair, error)
	Logout(ctx context.Context, jti string) error
	RefreshIDToken(ctx context.Context, userID int64, jti string) (*domain.TokenPair, error)
}

func (i *UserInteractor) GetByID(ctx context.Context, id int64) (*domain.User, error) {

	ctx, cancel := context.WithTimeout(ctx, i.ContextTimeout)
	defer cancel()

	res, err := i.UserRepository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (i *UserInteractor) Create(ctx context.Context, user *domain.User) (*domain.TokenPair, error) {
	ctx, cancel := context.WithTimeout(ctx, i.ContextTimeout)
	defer cancel()
	encryptedPass, err := genEncryptedPass(user.Password)
	if err != nil {
		return nil, err
	}
	user.EncryptedPassword = encryptedPass
	if err := i.UserRepository.Create(ctx, user); err != nil {
		return nil, err
	}
	tokenPair, err := i.TokenInteractor.GenerateTokenPair(user.ID)
	if err != nil {
		return nil, err
	}
	return tokenPair, nil
}

func (i *UserInteractor) Update(ctx context.Context, user *domain.User) error {
	ctx, cancel := context.WithTimeout(ctx, i.ContextTimeout)
	defer cancel()
	encryptedPass, err := genEncryptedPass(user.Password)
	if err != nil {
		return err
	}
	user.EncryptedPassword = encryptedPass
	if err := i.UserRepository.Update(ctx, user); err != nil {
		return err
	}
	return nil
}

func (i *UserInteractor) Delete(ctx context.Context, id int64) error {
	ctx, cancel := context.WithTimeout(ctx, i.ContextTimeout)
	defer cancel()

	return i.UserRepository.Delete(ctx, id)
}

func (i *UserInteractor) Login(ctx context.Context, email string, pass string) (*domain.User, *domain.TokenPair, error) {
	ctx, cancel := context.WithTimeout(ctx, i.ContextTimeout)
	defer cancel()
	user, err := i.UserRepository.GetByEmail(ctx, email)
	if err != nil {
		return nil, nil, err
	}
	if err := compareHashAndPass(user.EncryptedPassword, pass); err != nil {
		return nil, nil, err
	}
	tokenPair, err := i.TokenInteractor.GenerateTokenPair(user.ID)
	if err != nil {
		return nil, nil, err
	}

	return user, tokenPair, nil
}

func (i *UserInteractor) Logout(ctx context.Context, jti string) error {
	ctx, cancel := context.WithTimeout(ctx, i.ContextTimeout)
	defer cancel()
	return i.BlackListRepository.SAdd(jti)
}

func (i *UserInteractor) RefreshIDToken(ctx context.Context, userID int64, jti string) (*domain.TokenPair, error) {
	ctx, cancel := context.WithTimeout(ctx, i.ContextTimeout)
	defer cancel()
	blackListed, err := i.BlackListRepository.SIsMember(jti)
	if err != nil {
		return nil, err
	}
	if blackListed {
		return nil, status.Error(codes.Unauthenticated, "Token is blacklisted")
	}
	return i.TokenInteractor.GenerateTokenPair(userID)
}
