package auth

import (
	"context"
	"fmt"
	"testing"

	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"github.com/anthonymartz17/thinkmartz_backend/internal/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


func TestRepository_Save_Success(t *testing.T){
	// 1. Arrange: set up the real dependency
	       ctx:= t.Context()
         cfg,err:= config.Load()
         require.NoError(t,err,"configuration failed to load")
				 
				 pool,err:= database.NewPool(ctx,cfg)
         require.NoError(t,err,"Failed to create database pool")

				 repo:= NewRepository(pool)
				 email:= fmt.Sprintf("%sfake@email.com",t.Name())
			   user:= &User{
					Email: email,
					Username: t.Name(),
					PasswordHash:"fake-hashed-password",
				 }

	// 2. Act: call the method you're testing
	   gotErr:= repo.Save(ctx,user)

	// 3. Assert: check the result is what you expect
	assert.NoError(t,gotErr,"gotErr should be nil")
	assert.NotEqual(t,uuid.Nil,user.ID,"ID should not be UUID zero value")
	assert.False(t,user.CreatedAt.IsZero(),"Timestamp should not be zero")

  
	t.Cleanup(func() {
	_, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, user.ID)
	if err != nil {
		t.Logf("cleanup failed: %v", err)
	}
})
}
