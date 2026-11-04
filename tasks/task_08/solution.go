package main

import (
	"sync"
	"time"
)

type Clock interface {
	Now() time.Time
}

type Limiter struct {
	mu     sync.Mutex
	clock  Clock
	rate   float64
	burst  int
	tokens float64
	last   time.Time
}

func NewLimiter(clock Clock, ratePerSec float64, burst int) *Limiter {
	l := &Limiter{
		clock: clock,
		rate:  ratePerSec,
		burst: burst,
	}

	if clock == nil || burst <= 0 {
		return l
	}

	l.tokens = float64(burst)
	l.last = clock.Now()

	return l
}

func (l *Limiter) AllowN(n int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Проверяем корректность запроса.
	if l.clock == nil || l.burst <= 0 {
		return false
	}

	if n <= 0 || n > l.burst {
		return false
	}

	// Получаем текущее время.
	now := l.clock.Now()

	// Пополняем корзину, если время движется вперёд.
	if now.After(l.last) {
		elapsed := now.Sub(l.last).Seconds()

		if l.rate > 0 {
			l.tokens += elapsed * l.rate

			// Не превышаем максимальную вместимость.
			if l.tokens > float64(l.burst) {
				l.tokens = float64(l.burst)
			}
		}

		l.last = now
	}

	// Проверяем наличие необходимого количества токенов.
	if l.tokens < float64(n) {
		return false
	}

	// Атомарно списываем токены.
	l.tokens -= float64(n)

	return true
}
