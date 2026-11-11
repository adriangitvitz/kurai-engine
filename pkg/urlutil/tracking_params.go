package urlutil

var TrackingParamExact = func() map[string]struct{} {
	keys := []string{
		"utm_id", "utm_source", "utm_medium", "utm_campaign", "utm_term", "utm_content",
		"utm_brand", "utm_referrer", "_ga", "_gl",
		"gclid", "gbraid", "wbraid", "dclid", "msclkid", "fbclid",
		"mc_cid", "mc_eid", "vero_id", "vero_conv", "_hsenc", "_hsmi",
		"hsCtaTracking", "hsCta",
		"yclid", "ysclid", "igshid", "mkt_tok",
		"ref", "ref_", "ref_src",
		"oly_anon_id", "oly_enc_id",
		"ck_subscriber_id",
		"ICID", "NCID", "cmpid", "partnerid", "affiliate", "s_cid",
	}
	m := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		m[k] = struct{}{}
	}
	return m
}()

var TrackingParamPrefixes = []string{
	"utm_",
	"hsa_",
	"sc_",
}

func IsTrackingParam(name string) bool {
	if _, ok := TrackingParamExact[name]; ok {
		return true
	}
	for _, p := range TrackingParamPrefixes {
		if len(name) >= len(p) && name[:len(p)] == p {
			return true
		}
	}
	return false
}
