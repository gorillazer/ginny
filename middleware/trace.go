package middleware

import (
	"git.code.oa.com/linyyyang/ginny/logger"
	"git.code.oa.com/linyyyang/ginny/trace"
	"github.com/gin-gonic/gin"
)

// Trace 链路跟踪
func Trace() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reqID := ctx.GetHeader(trace.KeyReqID)
		if len(reqID) <= 0 {
			reqID = trace.RandReqID()
		}
		username := ctx.GetHeader(trace.KeyUsername)
		deviceID := ctx.GetHeader(trace.KeyDeviceID)

		msg := trace.GinMessage(ctx)
		o := trace.WithReqID(reqID)
		o(msg)
		o = trace.WithUsername(username)
		o(msg)
		o = trace.WithDeviceID(deviceID)
		o(msg)

		if msg.Logger != nil {
			o = trace.WithLogger(msg.Logger.With(msg.TraceFields()...))
			o(msg)
		} else {
			o = trace.WithLogger(logger.DefaultLogger.With(msg.TraceFields()...))
			o(msg)
		}
		ctx.Next()
	}
}