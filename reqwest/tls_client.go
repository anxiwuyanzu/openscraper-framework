package reqwest

import (
	"context"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/dot"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/reqwest/stats"
	http "github.com/bogdanfinn/fhttp"
	"github.com/bogdanfinn/fhttp/http2"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	tls "github.com/bogdanfinn/utls"
	"github.com/sirupsen/logrus"
	goproxy "golang.org/x/net/proxy"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type TlsClient struct {
	sync.Mutex
	*ProxyGuard

	innerClient        tls_client.HttpClient
	logger             *logrus.Entry
	option             *Option
	httpFailedTimes    int32
	CustomRedirectFunc func(req *http.Request, via []*http.Request) error
}

type DialerFunc struct {
	dialFunc DialFunc
}

func (df *DialerFunc) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return df.dialFunc(ctx, network, address)
}

func NewContextDialer(dialFunc DialFunc) goproxy.ContextDialer {
	return &DialerFunc{dialFunc: dialFunc}
}

func NewTlsClient(option *Option, opts ...OpOption) *TlsClient {
	if option == nil {
		option = DefaultOption()
	}
	for _, opFunc := range opts {
		opFunc(option)
	}
	logger := dot.Logger()

	f := &TlsClient{
		logger:     logger,
		option:     option,
		ProxyGuard: NewProxyGuard(option),
	}

	f.createClient()
	return f
}

func (c *TlsClient) createClient() {
	c.Lock()
	defer c.Unlock()

	dialer := c.getDialer()

	if len(c.option.ChromeVersion) == 0 {
		c.option.ChromeVersion = "130"
	}

	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(int(c.option.Timeout.Seconds())),
		tls_client.WithClientProfile(getTlsProfile(c.option.ChromeVersion)),
		//tls_client.WithCookieJar(tls_client.NewCookieJar()),
		tls_client.WithCustomRedirectFunc(c.customRedirectFunc),
		tls_client.WithTransportOptions(&tls_client.TransportOptions{
			ReadBufferSize:  c.option.ReadBufferSize,
			MaxConnsPerHost: c.option.MaxConnsPerHost,
		}),
		tls_client.WithProxyDialerFactory(func(proxyUrlStr string, timeout time.Duration, localAddr *net.TCPAddr, connectHeaders http.Header, logger tls_client.Logger) (goproxy.ContextDialer, error) {
			return NewContextDialer(dialer), nil
		}),
	}
	c.innerClient, _ = tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)

	c.closeIdleConnections = c.innerClient.CloseIdleConnections
}

func (c *TlsClient) customRedirectFunc(req *http.Request, via []*http.Request) error {
	if c.CustomRedirectFunc != nil {
		return c.CustomRedirectFunc(req, via)
	}
	return nil
}

func (c *TlsClient) Do(req *http.Request) (*http.Response, error) {
	c.option.Acquire()
	defer c.option.Release()

	c.CheckProxyAndCloseIdleConn()

	return c.innerClient.Do(req)
}

func (c *TlsClient) DoRequestTimeout(req IRequest, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	res := req.(ITlsRequest)
	httpReq := res.GetRequest()
	httpReq = httpReq.WithContext(ctx)
	resp, err := c.Do(httpReq)
	if err != nil {
		atomic.AddInt32(&c.httpFailedTimes, 1)
		stats.IncrRequestFailed()
		return err
	}

	atomic.StoreInt32(&c.httpFailedTimes, 0)
	return res.SetResponse(resp)
}

func (c *TlsClient) DoRequest(req IRequest) error {
	return c.DoRequestTimeout(req, c.option.Timeout)
}

func (c *TlsClient) DoRequestTimeoutAndRetry(req IRequest, timeout time.Duration, times int) error {
	var err error
	for times > 0 {
		times = times - 1
		err = c.DoRequestTimeout(req, timeout)
		if err == nil {
			return nil
		}
	}
	return err
}

func (c *TlsClient) InnerClient() tls_client.HttpClient {
	return c.innerClient
}

func (c *TlsClient) HttpFailedTimes() int {
	return int(c.httpFailedTimes)
}

func getTlsProfile(version string) profiles.ClientProfile {
	ver, _ := strconv.Atoi(version)
	if ver <= 130 {
		return profiles.NewClientProfile(tls.ClientHelloID{
			Client:               "Chrome",
			RandomExtensionOrder: false,
			Version:              "131",
			Seed:                 nil,
			SpecFactory: func() (tls.ClientHelloSpec, error) {
				clientHello := tls.ClientHelloSpec{
					CipherSuites: []uint16{
						tls.GREASE_PLACEHOLDER,
						tls.TLS_AES_128_GCM_SHA256,
						tls.TLS_AES_256_GCM_SHA384,
						tls.TLS_CHACHA20_POLY1305_SHA256,
						tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
						tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
						tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
						tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
						tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
						tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
						tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
						tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_RSA_WITH_AES_128_CBC_SHA,
						tls.TLS_RSA_WITH_AES_256_CBC_SHA,
					},
					CompressionMethods: []byte{
						tls.CompressionNone,
					},
					Extensions: []tls.TLSExtension{
						&tls.UtlsGREASEExtension{},
						&tls.SignatureAlgorithmsExtension{SupportedSignatureAlgorithms: []tls.SignatureScheme{
							tls.ECDSAWithP256AndSHA256,
							tls.PSSWithSHA256,
							tls.PKCS1WithSHA256,
							tls.ECDSAWithP384AndSHA384,
							tls.PSSWithSHA384,
							tls.PKCS1WithSHA384,
							tls.PSSWithSHA512,
							tls.PKCS1WithSHA512,
						}},
						tls.BoringGREASEECH(),
						&tls.RenegotiationInfoExtension{
							Renegotiation: tls.RenegotiateOnceAsClient,
						},
						&tls.SCTExtension{},
						&tls.UtlsCompressCertExtension{Algorithms: []tls.CertCompressionAlgo{
							tls.CertCompressionBrotli,
						}},
						&tls.ALPNExtension{AlpnProtocols: []string{
							"h2",
							"http/1.1",
						}},
						&tls.StatusRequestExtension{},
						&tls.SupportedCurvesExtension{Curves: []tls.CurveID{
							tls.GREASE_PLACEHOLDER,
							tls.X25519MLKEM768,
							tls.X25519,
							tls.CurveP256,
							tls.CurveP384,
						}},
						&tls.ApplicationSettingsExtension{
							CodePoint:          tls.ExtensionALPSOld,
							SupportedProtocols: []string{"h2"},
						},
						&tls.SupportedPointsExtension{SupportedPoints: []byte{
							tls.PointFormatUncompressed,
						}},
						&tls.KeyShareExtension{KeyShares: []tls.KeyShare{
							{Group: tls.CurveID(tls.GREASE_PLACEHOLDER), Data: []byte{0}},
							{Group: tls.X25519MLKEM768},
							{Group: tls.X25519},
						}},
						&tls.SessionTicketExtension{},
						&tls.SupportedVersionsExtension{Versions: []uint16{
							tls.GREASE_PLACEHOLDER,
							tls.VersionTLS13,
							tls.VersionTLS12,
						}},
						&tls.PSKKeyExchangeModesExtension{Modes: []uint8{
							tls.PskModeDHE,
						}},
						&tls.SNIExtension{},
						&tls.ExtendedMasterSecretExtension{},
						&tls.UtlsGREASEExtension{},
					},
				}
				clientHello.Extensions = randomizeExtensions(clientHello.Extensions)
				return clientHello, nil
			},
		}, map[http2.SettingID]uint32{
			http2.SettingHeaderTableSize:   65536,
			http2.SettingEnablePush:        0,
			http2.SettingInitialWindowSize: 6291456,
			http2.SettingMaxHeaderListSize: 262144,
		}, []http2.SettingID{
			http2.SettingHeaderTableSize,
			http2.SettingEnablePush,
			http2.SettingInitialWindowSize,
			http2.SettingMaxHeaderListSize,
		}, []string{
			":method",
			":authority",
			":scheme",
			":path",
		}, 15663105, []http2.Priority{}, &http2.PriorityParam{})
	}
	return profiles.Chrome_131
}

func randomizeExtensions(extensions []tls.TLSExtension) []tls.TLSExtension {
	// 创建一个新的切片，包含所有扩展
	randomExt := make([]tls.TLSExtension, len(extensions))
	copy(randomExt, extensions)

	// 完全随机打乱所有扩展的顺序
	rand.Shuffle(len(randomExt), func(i, j int) {
		randomExt[i], randomExt[j] = randomExt[j], randomExt[i]
	})

	return randomExt
}
