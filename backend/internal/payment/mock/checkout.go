package mock

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/logger"
)

func (g *Gateway) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+g.basePath+"/checkout", g.page)
	mux.HandleFunc("POST "+g.basePath+"/checkout/{ref}", g.submit)
	return mux
}

type checkoutView struct {
	Txn       *Txn
	Amount    string
	Action    string
	ReturnURL string
	Error     string
}

var checkoutTemplate = template.Must(template.New("checkout").Parse(`<!doctype html>
<html lang="vi"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Cổng thanh toán thử nghiệm</title>
<style>
body{font-family:system-ui,Arial,sans-serif;background:#f6f8fa;margin:0;padding:24px 16px;color:#1f2328}
.card{max-width:460px;margin:0 auto;background:#fff;border:1px solid #d0d7de;border-radius:8px;padding:24px}
.muted{color:#59636e;font-size:13px}
.error{background:#ffebe9;padding:10px 12px;border-radius:6px}
button,a.button{display:block;box-sizing:border-box;width:100%;padding:10px;margin-top:8px;border-radius:6px;border:1px solid #d0d7de;background:#f6f8fa;cursor:pointer;font-size:14px;text-align:center;color:inherit;text-decoration:none}
.primary{background:#1f883d !important;color:#fff !important;border-color:#1f883d !important}
code{word-break:break-all}
</style></head><body><div class="card">
<h2>Cổng thanh toán thử nghiệm</h2>
<p class="muted">Cổng giả lập (mock): không có tiền thật. Cổng gửi IPN và chuyển về cửa hàng giống một cổng thật.</p>
{{if .Error}}<p class="error">{{.Error}}</p>{{end}}
{{with .Txn}}
<p>{{.Description}}<br>Mã giao dịch: <code>{{.Ref}}</code><br>Số tiền: <b>{{$.Amount}}</b><br>Trạng thái: {{.State}}</p>
{{if eq .State "pending"}}
<form method="post" action="{{$.Action}}">
<button class="primary" name="mode" value="pay">Thanh toán</button>
<button name="mode" value="pay_no_ipn">Thanh toán, không gửi IPN (thử reconcile)</button>
<button name="mode" value="pay_wrong_amount">Thanh toán, cổng báo sai số tiền (thử hoàn tiền)</button>
<button name="mode" value="decline">Thẻ bị từ chối</button>
<button name="mode" value="cancel">Hủy thanh toán</button>
</form>
{{else if $.ReturnURL}}<a class="button primary" href="{{$.ReturnURL}}">Quay lại cửa hàng</a>{{end}}
{{end}}
</div></body></html>`))

func (g *Gateway) render(w http.ResponseWriter, status int, view checkoutView) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = checkoutTemplate.Execute(w, view)
}

func (g *Gateway) page(w http.ResponseWriter, r *http.Request) {
	t, ok := g.Lookup(r.URL.Query().Get("ref"))
	if !ok {
		g.render(w, http.StatusNotFound, checkoutView{
			Error: "Không tìm thấy giao dịch (cổng mock chỉ lưu trong bộ nhớ, khởi động lại server là mất).",
		})
		return
	}
	view := checkoutView{Txn: &t, Amount: formatAmount(t.Amount), Action: g.basePath + "/checkout/" + url.PathEscape(t.Ref)}
	if t.State != TxnPending {
		view.ReturnURL, _ = g.ReturnRedirect(t.Ref)
	}
	g.render(w, http.StatusOK, view)
}

func (g *Gateway) submit(w http.ResponseWriter, r *http.Request) {
	ref := r.PathValue("ref")
	mode := r.FormValue("mode")

	var err error
	switch mode {
	case "pay", "pay_no_ipn", "pay_wrong_amount", "decline":
		opts := CaptureOptions{Decline: mode == "decline"}
		if mode == "pay_wrong_amount" {
			opts.AmountDelta = 1000
		}
		if _, err = g.Capture(ref, opts); err == nil && mode != "pay_no_ipn" {
			if status, derr := g.Deliver(r.Context(), ref); derr != nil || status >= http.StatusMultipleChoices {
				logger.Warn("mock gateway: IPN not accepted by merchant",
					logger.String("ref", ref), logger.Int("status", status), logger.Err(derr))
			}
		}
	case "cancel":
		_, err = g.Cancel(ref)
	default:
		err = fmt.Errorf("unknown mode %q", mode)
	}
	if err != nil {
		g.render(w, http.StatusConflict, checkoutView{Error: err.Error()})
		return
	}

	target, err := g.ReturnRedirect(ref)
	if err != nil {
		g.render(w, http.StatusInternalServerError, checkoutView{Error: err.Error()})
		return
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

// formatAmount renders 120000 as "120.000 ₫".
func formatAmount(amount int64) string {
	digits := strconv.FormatInt(amount, 10)
	out := make([]byte, 0, len(digits)+len(digits)/3)
	for i := range len(digits) {
		if i > 0 && (len(digits)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, digits[i])
	}
	return string(out) + " ₫"
}
