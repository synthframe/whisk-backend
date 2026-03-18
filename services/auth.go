package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	hash  string
}

type AuthService struct {
	db      *pgxpool.Pool
	secret  string
	// in-memory fallback
	byEmail map[string]*User
	byID    map[string]*User
	mu      sync.RWMutex
}

func NewAuthService(db *pgxpool.Pool, secret string) *AuthService {
	return &AuthService{
		db:      db,
		secret:  secret,
		byEmail: make(map[string]*User),
		byID:    make(map[string]*User),
	}
}

func (a *AuthService) Register(email, password, name string) (string, *User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", nil, err
	}

	if a.db != nil {
		var user User
		err = a.db.QueryRow(context.Background(),
			`INSERT INTO users (email, name, password_hash) VALUES ($1, $2, $3)
			 RETURNING id, email, name`,
			email, name, string(hash),
		).Scan(&user.ID, &user.Email, &user.Name)
		if err != nil {
			if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
				return "", nil, errors.New("이미 등록된 이메일입니다")
			}
			return "", nil, err
		}
		token, err := a.generateToken(user.ID)
		return token, &user, err
	}

	// in-memory fallback
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, exists := a.byEmail[email]; exists {
		return "", nil, errors.New("이미 등록된 이메일입니다")
	}
	user := &User{
		ID:    uuid.New().String(),
		Email: email,
		Name:  name,
		hash:  string(hash),
	}
	a.byEmail[email] = user
	a.byID[user.ID] = user
	token, err := a.generateToken(user.ID)
	return token, user, err
}

func (a *AuthService) Login(email, password string) (string, *User, error) {
	if a.db != nil {
		var user User
		var hash string
		err := a.db.QueryRow(context.Background(),
			`SELECT id, email, name, password_hash FROM users WHERE email = $1`,
			email,
		).Scan(&user.ID, &user.Email, &user.Name, &hash)
		if err != nil {
			return "", nil, errors.New("이메일 또는 비밀번호가 올바르지 않습니다")
		}
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
			return "", nil, errors.New("이메일 또는 비밀번호가 올바르지 않습니다")
		}
		token, err := a.generateToken(user.ID)
		return token, &user, err
	}

	// in-memory fallback
	a.mu.RLock()
	user, exists := a.byEmail[email]
	a.mu.RUnlock()
	if !exists {
		return "", nil, errors.New("이메일 또는 비밀번호가 올바르지 않습니다")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.hash), []byte(password)); err != nil {
		return "", nil, errors.New("이메일 또는 비밀번호가 올바르지 않습니다")
	}
	token, err := a.generateToken(user.ID)
	return token, user, err
}

func (a *AuthService) GetUser(id string) (*User, error) {
	if a.db != nil {
		var user User
		err := a.db.QueryRow(context.Background(),
			`SELECT id, email, name FROM users WHERE id = $1`, id,
		).Scan(&user.ID, &user.Email, &user.Name)
		if err != nil {
			return nil, errors.New("사용자를 찾을 수 없습니다")
		}
		return &user, nil
	}

	// in-memory fallback
	a.mu.RLock()
	defer a.mu.RUnlock()
	user, exists := a.byID[id]
	if !exists {
		return nil, fmt.Errorf("사용자를 찾을 수 없습니다")
	}
	return user, nil
}

func (a *AuthService) ValidateToken(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(a.secret), nil
	})
	if err != nil || !token.Valid {
		return "", errors.New("유효하지 않은 토큰입니다")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid claims")
	}
	userID, ok := claims["sub"].(string)
	if !ok {
		return "", errors.New("invalid sub")
	}
	return userID, nil
}

func (a *AuthService) generateToken(userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	return token.SignedString([]byte(a.secret))
}
