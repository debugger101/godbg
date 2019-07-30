package log

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net/http"
	"os"
)

var Log *zap.Logger

func init() {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "ts"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.CallerKey = "caller"
	encoderCfg.EncodeCaller = zapcore.ShortCallerEncoder

	level := zapcore.InfoLevel

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
