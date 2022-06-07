package root

import (
	"net/http"

	"github.com/zecrey-labs/zecrey-legend/service/api/explorer/internal/logic/root"
	"github.com/zecrey-labs/zecrey-legend/service/api/explorer/internal/svc"
	"github.com/zecrey-labs/zecrey-legend/service/api/explorer/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ReqGetStatus
		if err := httpx.Parse(r, &req); err != nil {
			httpx.Error(w, err)
			return
		}

		l := root.NewGetStatusLogic(r.Context(), svcCtx)
		resp, err := l.GetStatus(&req)
		if err != nil {
			httpx.Error(w, err)
		} else {
			httpx.OkJson(w, resp)
		}
	}
}