package gapi

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v5/pgtype"
	mockdb "github.com/sowmiyaramanathan/A-simple-bank-app-using-Golang/db/mock"
	db "github.com/sowmiyaramanathan/A-simple-bank-app-using-Golang/db/sqlc"
	"github.com/sowmiyaramanathan/A-simple-bank-app-using-Golang/db/util"
	pb "github.com/sowmiyaramanathan/A-simple-bank-app-using-Golang/pb"
	"github.com/sowmiyaramanathan/A-simple-bank-app-using-Golang/token"
	"github.com/sowmiyaramanathan/A-simple-bank-app-using-Golang/worker"
	mockwk "github.com/sowmiyaramanathan/A-simple-bank-app-using-Golang/worker/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type eqCreateUserTxParamsMatcher struct {
	arg      db.CreateUserTxParams
	password string
	user     db.User
}

func (expected eqCreateUserTxParamsMatcher) Matches(x interface{}) bool {
	actualArg, ok := x.(db.CreateUserTxParams)
	if !ok {
		return false
	}

	err := util.CheckPassword(expected.password, actualArg.HashedPassword)
	if err != nil {
		return false
	}

	expected.arg.HashedPassword = actualArg.HashedPassword
	if !reflect.DeepEqual(expected.arg.CreateUserParams, actualArg.CreateUserParams) {
		return false
	}

	err = actualArg.AfterCreate(expected.user)
	return err == nil
}

func (e eqCreateUserTxParamsMatcher) String() string {
	return fmt.Sprintf("matches arg %v and password %v", e.arg, e.password)
}

func EqCreateUserTxParams(arg db.CreateUserTxParams, password string, user db.User) gomock.Matcher {
	return eqCreateUserTxParamsMatcher{arg, password, user}
}

func randomUser(t *testing.T) (db.User, string) {
	password := util.RandomString(6)
	hashedPassword, err := util.HashPassword(password)
	require.NoError(t, err)

	user := db.User{
		Username:       util.RandomOwner(),
		Role:           util.DepositorRole,
		HashedPassword: hashedPassword,
		FullName:       util.RandomOwner(),
		Email:          util.RandomEmail(),
	}

	return user, password
}

func TestCreateUserAPI(t *testing.T) {
	user, password := randomUser(t)

	testCases := []struct {
		name          string
		req           *pb.CreateUserRequest
		buildStubs    func(store *mockdb.MockStore, taskDistributor *mockwk.MockTaskDistributor)
		checkResponse func(t *testing.T, res *pb.CreateUserResponse, err error)
	}{
		{
			name: "OK",
			req: &pb.CreateUserRequest{
				Username: user.Username,
				Password: password,
				FullName: user.FullName,
				Email:    user.Email,
			},
			buildStubs: func(store *mockdb.MockStore, taskDistributor *mockwk.MockTaskDistributor) {
				arg := db.CreateUserTxParams{
					CreateUserParams: db.CreateUserParams{
						Username: user.Username,
						FullName: user.FullName,
						Email:    user.Email,
					},
				}
				store.EXPECT().CreateUserTx(gomock.Any(), EqCreateUserTxParams(arg, password, user)).Times(1).Return(db.CreateUserTxResult{User: user}, nil)
				taskPayload := &worker.PayloadSendVerifyEmail{
					Username: user.Username,
				}
				taskDistributor.EXPECT().DistributeTaskSendVerifyEmail(gomock.Any(), taskPayload, gomock.Any()).Times(1).Return(nil)
			},
			checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
				require.NoError(t, err)
				require.NotNil(t, res)
				createdUser := res.GetUser()
				require.Equal(t, user.Username, createdUser.Username)
				require.Equal(t, user.FullName, createdUser.FullName)
				require.Equal(t, user.Email, createdUser.Email)
			},
		},
		{
			name: "InternalError",
			req: &pb.CreateUserRequest{
				Username: user.Username,
				Password: password,
				FullName: user.FullName,
				Email:    user.Email,
			},
			buildStubs: func(store *mockdb.MockStore, taskDistributor *mockwk.MockTaskDistributor) {
				store.EXPECT().CreateUserTx(gomock.Any(), gomock.Any()).Times(1).Return(db.CreateUserTxResult{}, sql.ErrConnDone)
				taskDistributor.EXPECT().DistributeTaskSendVerifyEmail(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.Internal, st.Code())
			},
		},
		// {
		// 	name: "DuplicateUsername",
		// 	req: &pb.CreateUserRequest{
		// 		Username: user.Username,
		// 		Password: password,
		// 		FullName: user.FullName,
		// 		Email:    user.Email,
		// 	},
		// 	buildStubs: func(store *mockdb.MockStore, taskDistributor *mockwk.MockTaskDistributor) {
		// 		store.EXPECT().
		// 			CreateUserTx(gomock.Any(), gomock.Any()).
		// 			Times(1).
		// 			Return(db.CreateUserTxResult{}, db.ErrUniqueViolation)

		// 		taskDistributor.EXPECT().
		// 			DistributeTaskSendVerifyEmail(gomock.Any(), gomock.Any(), gomock.Any()).
		// 			Times(0)
		// 	},
		// 	checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
		// 		require.Error(t, err)
		// 		st, ok := status.FromError(err)
		// 		require.True(t, ok)
		// 		require.Equal(t, codes.AlreadyExists, st.Code())
		// 	},
		// },
		// {
		// 	name: "InvalidEmail",
		// 	req: &pb.CreateUserRequest{
		// 		Username: user.Username,
		// 		Password: password,
		// 		FullName: user.FullName,
		// 		Email:    "invalid-email",
		// 	},
		// 	buildStubs: func(store *mockdb.MockStore, taskDistributor *mockwk.MockTaskDistributor) {
		// 		store.EXPECT().
		// 			CreateUserTx(gomock.Any(), gomock.Any()).
		// 			Times(0)

		// 		taskDistributor.EXPECT().
		// 			DistributeTaskSendVerifyEmail(gomock.Any(), gomock.Any(), gomock.Any()).
		// 			Times(0)
		// 	},
		// 	checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
		// 		require.Error(t, err)
		// 		st, ok := status.FromError(err)
		// 		require.True(t, ok)
		// 		require.Equal(t, codes.InvalidArgument, st.Code())
		// 	},
		// },
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			storeCtrl := gomock.NewController(t)
			defer storeCtrl.Finish()
			store := mockdb.NewMockStore(storeCtrl)

			taskCtrl := gomock.NewController(t)
			defer taskCtrl.Finish()
			taskDistributor := mockwk.NewMockTaskDistributor(taskCtrl)

			tc.buildStubs(store, taskDistributor)
			server := newTestServer(t, store, taskDistributor)

			res, err := server.CreateUser(context.Background(), tc.req)
			tc.checkResponse(t, res, err)
		})
	}
}

func TestUpdateUserAPI(t *testing.T) {
	user, _ := randomUser(t)

	newName := util.RandomOwner()
	newEmail := util.RandomEmail()
	invalidEmail := "invalid-email"

	testCases := []struct {
		name          string
		req           *pb.UpdateUserRequest
		buildStubs    func(store *mockdb.MockStore)
		buildContext  func(t *testing.T, tokenMaker token.Maker) context.Context
		checkResponse func(t *testing.T, res *pb.UpdateUserResponse, err error)
	}{
		{
			name: "OK",
			req: &pb.UpdateUserRequest{
				Username: user.Username,
				FullName: &newName,
				Email:    &newEmail,
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.UpdateUserParams{
					Username: user.Username,
					FullName: pgtype.Text{
						String: newName,
						Valid:  true,
					},
					Email: pgtype.Text{
						String: newEmail,
						Valid:  true,
					},
				}
				updatedUser := db.User{
					Username:          user.Username,
					HashedPassword:    user.HashedPassword,
					FullName:          newName,
					Email:             newEmail,
					PasswordChangedAt: user.PasswordChangedAt,
					CreatedAt:         user.CreatedAt,
					IsEmailVerified:   user.IsEmailVerified,
				}
				store.EXPECT().UpdateUser(gomock.Any(), gomock.Eq(arg)).Times(1).Return(updatedUser, nil)
			},
			buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
				return newContextWithBearerToken(t, tokenMaker, user.Username, user.Role, time.Minute)
			},
			checkResponse: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
				require.NoError(t, err)
				require.NotNil(t, res)
				updatedUser := res.GetUser()
				require.Equal(t, user.Username, updatedUser.Username)
				require.Equal(t, newName, updatedUser.FullName)
				require.Equal(t, newEmail, updatedUser.Email)
			},
		},
		// {
		// 	name: "BankerCanUpdateUserInfo",
		// 	req: &pb.UpdateUserRequest{
		// 		Username: user.Username,
		// 		FullName: &newName,
		// 		Email:    &newEmail,
		// 	},
		// 	buildStubs: func(store *mockdb.MockStore) {
		// 		arg := db.UpdateUserParams{
		// 			Username: user.Username,
		// 			FullName: pgtype.Text{
		// 				String: newName,
		// 				Valid:  true,
		// 			},
		// 			Email: pgtype.Text{
		// 				String: newEmail,
		// 				Valid:  true,
		// 			},
		// 		}
		// 		updatedUser := db.User{
		// 			Username:          user.Username,
		// 			HashedPassword:    user.HashedPassword,
		// 			FullName:          newName,
		// 			Email:             newEmail,
		// 			PasswordChangedAt: user.PasswordChangedAt,
		// 			CreatedAt:         user.CreatedAt,
		// 			IsEmailVerified:   user.IsEmailVerified,
		// 		}
		// 		store.EXPECT().
		// 			UpdateUser(gomock.Any(), gomock.Eq(arg)).
		// 			Times(1).
		// 			Return(updatedUser, nil)
		// 	},
		// 	buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
		// 		return newContextWithBearerToken(t, tokenMaker, banker.Username, banker.Role, time.Minute, token.TokenTypeAccessToken)
		// 	},
		// 	checkResponse: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
		// 		require.NoError(t, err)
		// 		require.NotNil(t, res)
		// 		updatedUser := res.GetUser()
		// 		require.Equal(t, user.Username, updatedUser.Username)
		// 		require.Equal(t, newName, updatedUser.FullName)
		// 		require.Equal(t, newEmail, updatedUser.Email)
		// 	},
		// },
		// {
		// 	name: "OtherDepositorCannotUpdateThisUserInfo",
		// 	req: &pb.UpdateUserRequest{
		// 		Username: user.Username,
		// 		FullName: &newName,
		// 		Email:    &newEmail,
		// 	},
		// 	buildStubs: func(store *mockdb.MockStore) {
		// 		store.EXPECT().
		// 			UpdateUser(gomock.Any(), gomock.Any()).
		// 			Times(0)
		// 	},
		// 	buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
		// 		return newContextWithBearerToken(t, tokenMaker, other.Username, other.Role, time.Minute, token.TokenTypeAccessToken)
		// 	},
		// 	checkResponse: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
		// 		require.Error(t, err)
		// 		st, ok := status.FromError(err)
		// 		require.True(t, ok)
		// 		require.Equal(t, codes.PermissionDenied, st.Code())
		// 	},
		// },
		{
			name: "UserNotFound",
			req: &pb.UpdateUserRequest{
				Username: user.Username,
				FullName: &newName,
				Email:    &newEmail,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUser(gomock.Any(), gomock.Any()).Times(1).Return(db.User{}, db.ErrRecordNotFound)
			},
			buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
				return newContextWithBearerToken(t, tokenMaker, user.Username, user.Role, time.Minute)
			},
			checkResponse: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.NotFound, st.Code())
			},
		},
		{
			name: "InvalidEmail",
			req: &pb.UpdateUserRequest{
				Username: user.Username,
				FullName: &newName,
				Email:    &invalidEmail,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
				return newContextWithBearerToken(t, tokenMaker, user.Username, user.Role, time.Minute)
			},
			checkResponse: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.InvalidArgument, st.Code())
			},
		},
		{
			name: "ExpiredToken",
			req: &pb.UpdateUserRequest{
				Username: user.Username,
				FullName: &newName,
				Email:    &newEmail,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUser(gomock.Any(), gomock.Any()).Times(0)
			},
			buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
				return newContextWithBearerToken(t, tokenMaker, user.Username, user.Role, -time.Minute)
			},
			checkResponse: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.Unauthenticated, st.Code())
			},
		},
		// {
		// 	name: "WrongTokenType",
		// 	req: &pb.UpdateUserRequest{
		// 		Username: user.Username,
		// 		FullName: &newName,
		// 		Email:    &newEmail,
		// 	},
		// 	buildStubs: func(store *mockdb.MockStore) {
		// 		store.EXPECT().
		// 			UpdateUser(gomock.Any(), gomock.Any()).
		// 			Times(0)
		// 	},
		// 	buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
		// 		return newContextWithBearerToken(t, tokenMaker, user.Username, user.Role, time.Minute, token.TokenTypeRefreshToken)
		// 	},
		// 	checkResponse: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
		// 		require.Error(t, err)
		// 		st, ok := status.FromError(err)
		// 		require.True(t, ok)
		// 		require.Equal(t, codes.Unauthenticated, st.Code())
		// 	},
		// },
		{
			name: "NoAuthorization",
			req: &pb.UpdateUserRequest{
				Username: user.Username,
				FullName: &newName,
				Email:    &newEmail,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUser(gomock.Any(), gomock.Any()).Times(0)
			},
			buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
				return context.Background()
			},
			checkResponse: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.Unauthenticated, st.Code())
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			storeCtrl := gomock.NewController(t)
			defer storeCtrl.Finish()
			store := mockdb.NewMockStore(storeCtrl)

			tc.buildStubs(store)
			server := newTestServer(t, store, nil)

			ctx := tc.buildContext(t, server.tokenMaker)
			res, err := server.UpdateUser(ctx, tc.req)
			tc.checkResponse(t, res, err)
		})
	}
}
