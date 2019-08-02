package log

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net/http"
	"os"
	"strings"
)

var Log *zap.Logger

func init() {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = ""
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.LevelKey = "lv"
	encoderCfg.CallerKey = "caller"
	encoderCfg.EncodeCaller = zapcore.ShortCallerEncoder

	logLv := strings.ToLower(os.Getenv("DBGLOGLV"))
	var level zapcore.Level

	switch logLv {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "panic":
		level = zapcore.PanicLevel
	default:
		level = zapcore.PanicLevel
	}


	http.HandleFunc("/debug", func(w http.ResponseWriter, r *http.Request) {
		lv := r.PostFormValue("level")
		if lv == "" {
			fmt.Fprintf(w, "%s\n", level.String())
			return
		}
		if err := level.Set(lv); err != nil {
			fmt.Fprintf(w, "err:%s, keep level=%s\n", err, level.String())
			return
		}
		fmt.Fprintf(w, "%s\n", level.String())
	})
	go func() {
		if err := http.ListenAndServe(":9090", nil); err != nil {
			panic(err)
		}
	}()

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		level,
	)
	Log = zap.New(core)
}
