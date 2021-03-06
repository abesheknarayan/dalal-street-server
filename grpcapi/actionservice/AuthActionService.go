package actionservice

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/delta/dalal-street-server/models"
	actions_pb "github.com/delta/dalal-street-server/proto_build/actions"
	models_pb "github.com/delta/dalal-street-server/proto_build/models"
	"github.com/delta/dalal-street-server/session"
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

func writeUserDetailsToLog(ctx context.Context) {
	var l = logger.WithFields(logrus.Fields{
		"method": "writeUserDetailsToLog",
	})

	userID := getUserId(ctx)

	peerDetails, ok := peer.FromContext(ctx)
	if ok {
		err := models.AddToGeneralLog(userID, "IP", peerDetails.Addr.String())
		if err != nil {
			l.Infof("Error while writing to databaes. Error: %+v", err)
		}
	} else {
		l.Infof("Failed to log peer details")
	}

	mD, ok := metadata.FromIncomingContext(ctx)
	if ok {
		userAgent := strings.Join(mD["user-agent"], " ")
		err := models.AddToGeneralLog(userID, "User-Agent", userAgent)
		if err != nil {
			l.Infof("Error while writing to databaes. Error: %+v", err)
		}
	} else {
		l.Infof("Failed to log user-agent")
	}
}

func (d *dalalActionService) Register(ctx context.Context, req *actions_pb.RegisterRequest) (*actions_pb.RegisterResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "Register",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})

	l.Infof("Register requested")

	resp := &actions_pb.RegisterResponse{}
	makeError := func(st actions_pb.RegisterResponse_StatusCode, msg string) (*actions_pb.RegisterResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	err := models.RegisterUser(req.GetEmail(), req.GetPassword(), req.GetFullName())
	if err == models.AlreadyRegisteredError {
		return makeError(actions_pb.RegisterResponse_AlreadyRegisteredError, "Already registered please Login")
	}
	if err != nil {
		l.Errorf("Request failed due to: %+v", err)
		return makeError(actions_pb.RegisterResponse_InternalServerError, getInternalErrorMessage(err))
	}

	l.Infof("Done")

	return resp, nil
}

func (d *dalalActionService) Login(ctx context.Context, req *actions_pb.LoginRequest) (*actions_pb.LoginResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "Login",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})

	l.Infof("Login requested")

	resp := &actions_pb.LoginResponse{}
	makeError := func(st actions_pb.LoginResponse_StatusCode, msg string) (*actions_pb.LoginResponse, error) {
		resp.StatusCode = st
		resp.StatusMessage = msg
		return resp, nil
	}

	var (
		user            models.User
		err             error
		alreadyLoggedIn bool
	)

	sess := ctx.Value("session").(session.Session)
	if userId, ok := sess.Get("userId"); !ok {
		email := req.GetEmail()
		password := req.GetPassword()

		if email == "" || password == "" {
			return makeError(actions_pb.LoginResponse_InvalidCredentialsError, "Invalid Credentials")
		}

		user, err = models.Login(email, password)
	} else {
		alreadyLoggedIn = true
		userIdInt, err := strconv.ParseUint(userId, 10, 32)
		if err == nil {
			user, err = models.GetUserCopy(uint32(userIdInt))
		}
	}

	switch {
	case err == models.UnauthorizedError:
		return makeError(actions_pb.LoginResponse_InvalidCredentialsError, "Incorrect username/password combination. Please use your Pragyan credentials.")
	case err == models.NotRegisteredError:
		return makeError(actions_pb.LoginResponse_InvalidCredentialsError, "You have not registered for Dalal Street on the Pragyan website")
	case err != nil:
		l.Errorf("Request failed due to: %+v", err)
		return makeError(actions_pb.LoginResponse_InternalServerError, getInternalErrorMessage(err))
	}

	l.Debugf("models.Login returned without error %+v", user)

	if !alreadyLoggedIn {
		if err := sess.Set("userId", strconv.Itoa(int(user.Id))); err != nil {
			l.Errorf("Request failed due to: %+v", err)
			return makeError(actions_pb.LoginResponse_InternalServerError, getInternalErrorMessage(err))
		}
	}

	writeUserDetailsToLog(ctx)

	l.Debugf("Session successfully set. UserId: %+v, Session id: %+v", user.Id, sess.GetID())

	stocksOwned, err := models.GetStocksOwned(user.Id)
	if err != nil {
		l.Errorf("Request failed due to %+v", err)
		return makeError(actions_pb.LoginResponse_InternalServerError, getInternalErrorMessage(err))
	}

	stockList := models.GetAllStocks()
	stockListProto := make(map[uint32]*models_pb.Stock)
	for stockId, stock := range stockList {
		stockListProto[stockId] = stock.ToProto()
	}

	constantsMap := map[string]int32{
		"SHORT_SELL_BORROW_LIMIT": models.SHORT_SELL_BORROW_LIMIT,
		"BID_LIMIT":               models.BID_LIMIT,
		"ASK_LIMIT":               models.ASK_LIMIT,
		"BUY_LIMIT":               models.BUY_LIMIT,
		"MINIMUM_CASH_LIMIT":      models.MINIMUM_CASH_LIMIT,
		"BUY_FROM_EXCHANGE_LIMIT": models.BUY_FROM_EXCHANGE_LIMIT,
		"ORDER_PRICE_WINDOW":      models.ORDER_PRICE_WINDOW,
		"STARTING_CASH":           models.STARTING_CASH,
		"MORTGAGE_RETRIEVE_RATE":  models.MORTGAGE_RETRIEVE_RATE,
		"MORTGAGE_DEPOSIT_RATE":   models.MORTGAGE_DEPOSIT_RATE,
		"MARKET_EVENT_COUNT":      models.MARKET_EVENT_COUNT,
		"MY_ASK_COUNT":            models.MY_ASK_COUNT,
		"MY_BID_COUNT":            models.MY_BID_COUNT,
		"GET_NOTIFICATION_COUNT":  models.GET_NOTIFICATION_COUNT,
		"GET_TRANSACTION_COUNT":   models.GET_TRANSACTION_COUNT,
		"LEADERBOARD_COUNT":       models.LEADERBOARD_COUNT,
		"ORDER_FEE_PERCENT":       models.ORDER_FEE_PERCENT,
	}

	reservedStocksOwned, err := models.GetReservedStocksOwned(user.Id)
	if err != nil {
		l.Errorf("Unable to get Reserved Stocks for User Id. Error: %+v", err)
		return makeError(actions_pb.LoginResponse_InternalServerError, getInternalErrorMessage(err))
	}

	resp = &actions_pb.LoginResponse{
		SessionId:                sess.GetID(),
		User:                     user.ToProto(),
		StocksOwned:              stocksOwned,
		StockList:                stockListProto,
		Constants:                constantsMap,
		IsMarketOpen:             models.IsMarketOpen(),
		MarketIsClosedHackyNotif: models.MARKET_IS_CLOSED_HACKY_NOTIF,
		MarketIsOpenHackyNotif:   models.MARKET_IS_OPEN_HACKY_NOTIF,
		ReservedStocksOwned:      reservedStocksOwned,
	}

	l.Infof("Request completed successfully")

	return resp, nil
}

func (d *dalalActionService) Logout(ctx context.Context, req *actions_pb.LogoutRequest) (*actions_pb.LogoutResponse, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":        "Logout",
		"param_session": fmt.Sprintf("%+v", ctx.Value("session")),
		"param_req":     fmt.Sprintf("%+v", req),
	})

	l.Infof("Logout requested")

	sess := ctx.Value("session").(session.Session)
	userId := getUserId(ctx)
	models.Logout(userId)
	sess.Destroy()

	l.Infof("Request completed successfully")

	return &actions_pb.LogoutResponse{}, nil
}
