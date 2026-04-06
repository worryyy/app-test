package com.jb.school.helper;

import com.google.common.collect.Lists;
import com.jb.common.utils.UserAgentGeneratorUtils;
import lombok.NonNull;
import lombok.extern.slf4j.Slf4j;
import okhttp3.*;
import okio.Buffer;
import org.apache.commons.lang3.StringUtils;
import org.apache.http.*;
import org.apache.http.client.HttpClient;
import org.apache.http.client.entity.UrlEncodedFormEntity;
import org.apache.http.client.methods.CloseableHttpResponse;
import org.apache.http.client.methods.HttpPost;
import org.apache.http.conn.ssl.NoopHostnameVerifier;
import org.apache.http.impl.client.CloseableHttpClient;
import org.apache.http.impl.client.HttpClientBuilder;
import org.apache.http.impl.client.HttpClients;
import org.apache.http.impl.client.LaxRedirectStrategy;
import org.apache.http.message.BasicHeader;
import org.apache.http.message.BasicNameValuePair;
import org.apache.http.protocol.HttpContext;
import org.apache.http.util.Args;
import org.springframework.context.annotation.Scope;
import org.springframework.stereotype.Component;

import javax.crypto.Cipher;
import javax.crypto.SecretKey;
import javax.crypto.SecretKeyFactory;
import javax.crypto.spec.DESKeySpec;
import java.io.IOException;
import java.net.CookieManager;
import java.net.CookiePolicy;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.Base64;
import java.util.List;
import java.util.Map;

@Slf4j
@Component
@Scope("prototype")
public class JWRequestHelper {

    private static final String agent = UserAgentGeneratorUtils.genChromeUA();

    private static final Header userAgent = new BasicHeader(HttpHeaders.USER_AGENT, agent);

    private static final Header host = new BasicHeader(HttpHeaders.HOST, "auth.sztu.edu.cn");
    private static final Header referer = new BasicHeader(HttpHeaders.REFERER, "https://auth.sztu.edu.cn/idp/authcenter/ActionAuthChain?entityId=jiaowu");
    private static final Header origin = new BasicHeader("Origin", "https://auth.sztu.edu.cn");
    private static final Header contentType = new BasicHeader(HttpHeaders.CONTENT_TYPE, "application/x-www-form-urlencoded; charset=UTF-8");

    private static final Header XReq = new BasicHeader("X-Requested-With", "XMLHttpRequest");

    private static final List<Header> headers = Lists.newArrayList(userAgent, host, referer, origin, contentType, XReq );


    private static String loginCookie = "";


    public String encryptByDES(String password) {
        String secretKey = "PassB01Il71".substring(0, 8); // 密钥
        try {
            DESKeySpec desKeySpec = new DESKeySpec(secretKey.getBytes(StandardCharsets.UTF_8));
            SecretKeyFactory keyFactory = SecretKeyFactory.getInstance("DES");
            SecretKey secretKeyObj = keyFactory.generateSecret(desKeySpec);

            Cipher cipher = Cipher.getInstance("DES/ECB/PKCS5Padding");
            cipher.init(Cipher.ENCRYPT_MODE, secretKeyObj);
            byte[] encryptedBytes = cipher.doFinal(password.getBytes(StandardCharsets.UTF_8));

            return Base64.getEncoder().encodeToString(encryptedBytes);
        } catch (Exception e) {
            log.error("获取加密后的密码出错, error->{}", e);
        }
        return null;
    }

    private void preClean() {
        loginCookie = "";
    }

    public String getLoginCookie() {
        return loginCookie;
    }

    public void loginGet(String url) {
        preClean();
        // 一定要构造 manager accept all cookie  使用okhttp3 进行请求获取cookie这就是玄学（因为要捕获重定向的cookie）
        CookieManager cookieManager = new CookieManager();
        cookieManager.setCookiePolicy(CookiePolicy.ACCEPT_ALL);
        OkHttpClient client = new OkHttpClient.Builder()
                .eventListener(new RequestEventListener())
                .followRedirects(true)
                .followSslRedirects(true)
                .cookieJar(new JavaNetCookieJar(cookieManager))
                .hostnameVerifier(new NoopHostnameVerifier())
                .build();
        Request request = new Request.Builder()
            .url(url)
            .header(userAgent.getName(), userAgent.getValue())
            .header(contentType.getName(), contentType.getValue())
            .build();
        try{
            client.newCall(request).execute();
        } catch (Exception e) {
            log.error("get {} occurs error:", url, e);
        }
    }
    public Response getWithCookie(String url, String cookie) {
        OkHttpClient client = new OkHttpClient.Builder()
                .eventListener(new RequestEventListener())
                .followRedirects(true)
                .followSslRedirects(true)
                .hostnameVerifier(new NoopHostnameVerifier())
                .build();
        Request.Builder builder = new Request.Builder()
                .url(url)
                .addHeader("User-Agent", agent)
                .addHeader(contentType.getName(), contentType.getValue())
                .addHeader("Cookie", cookie);
        for(Header h : headers) {
            builder.addHeader(h.getName(), h.getValue());
        }
        Request request = builder
                .build();
        try{
            return client.newCall(request).execute();
        } catch (Exception e) {
            log.error("get {} occurs error:", url, e);
        }
        return null;
    }

    public CloseableHttpResponse post(@NonNull String url, @NonNull Map<String, String> data, @NonNull String loginCookie) throws IOException {
        HttpClient client = HttpClientBuilder.create()
                //.setDefaultCookieStore(cookieStore)
                .addInterceptorLast(new RequestInterceptor())
                .addInterceptorLast(new ResponseInterceptor())
                .setDefaultHeaders(headers)
                .setUserAgent(agent)
                .build();
        // 准备登录参数
        List<NameValuePair> params = new ArrayList<>();
        data.forEach((k,v)->{
            params.add(new BasicNameValuePair(k, v));
        });


        UrlEncodedFormEntity entity = new UrlEncodedFormEntity(params, "UTF-8");
        // 发送登录请求
        HttpPost httpPost = new HttpPost(url);
        httpPost.addHeader("Cookie", loginCookie);
        httpPost.setEntity(entity);
        try {
            return (CloseableHttpResponse) client.execute(httpPost);
        } catch (Exception e) {
            log.error("post {} occurs error: {}", url, e.getMessage());
        }
        return null;
    }

    public CloseableHttpResponse postAllowRedirect(@NonNull String url, @NonNull Map<String, String> data, @NonNull String loginCookie) throws IOException {
        CloseableHttpClient client = HttpClients.custom()
                .setSSLHostnameVerifier(new NoopHostnameVerifier())
                .addInterceptorLast(new RequestInterceptor())
                .addInterceptorLast(new ResponseInterceptor())
                .setRedirectStrategy(new LaxRedirectStrategy())
//                .setRedirectStrategy(new CustomRedirectStrategy())
                 .setDefaultHeaders(headers)
                .build();
        // 准备登录参数
        List<NameValuePair> params = new ArrayList<>();
        data.forEach((k,v)->{
            params.add(new BasicNameValuePair(k, v));
        });

        UrlEncodedFormEntity entity = new UrlEncodedFormEntity(params, "UTF-8");
        // 发送登录请求
        HttpPost httpPost = new HttpPost(url);
        httpPost.setEntity(entity);
        httpPost.addHeader("Cookie", loginCookie);
        try {
            return client.execute(httpPost);
        } catch (Exception e) {
            log.error("post {} occurs error: {}", url, e.getMessage());
        } finally {
            log.info("<<<<<<<<<<<<<<<<< post client close");
            client.close();
        }
        return null;
    }




    private static class RequestInterceptor implements HttpRequestInterceptor {
        @Override
        public void process(HttpRequest request, HttpContext context) {
            System.out.println("Request Method: " + request.getRequestLine().getMethod());
            System.out.println("Request URI: " + request.getRequestLine().getUri());
            Header[] requestHeaders = request.getAllHeaders();
            for (Header header : requestHeaders) {
                System.out.println(header.getName() + ": " + header.getValue());
            }
            System.out.println("==========request end ===========");
        }
    }

    private static class ResponseInterceptor implements HttpResponseInterceptor {
        @Override
        public void process(HttpResponse response, HttpContext httpContext) throws HttpException, IOException {
            System.out.println("code: " + response.getStatusLine().getStatusCode());
            Header[] responseAllHeaders = response.getAllHeaders();
            for (Header header : responseAllHeaders) {
                System.out.println(header.getName() + ": " + header.getValue());
            }
            System.out.println("-------------response end ----------------");
        }
    }


    /**
     * 重写重定向，不获取大文件首页，
     */
    private static class CustomRedirectStrategy extends LaxRedirectStrategy {
        @Override
        public boolean isRedirected(HttpRequest request, HttpResponse response, HttpContext context) throws ProtocolException {
            try{
                Args.notNull(request, "HTTP request");
                Args.notNull(response, "HTTP response");
                Header locationHeader = response.getFirstHeader("location");
                if (locationHeader != null) {
                    log.info("location=>{}",locationHeader.getValue() );
                }

                if(locationHeader.getValue().contains("https://jwxt.sztu.edu.cn/jsxsd/framework/xsMain.htmlx")){
                    log.info("get main.html and has block by customRedirectStrategy!!!");
                    return false;
                }
                return super.isRedirected(request, response, context);
            } catch (Exception e) {
                log.error("自定义重定向策略错误: e={}", e.getMessage());
            }
            return false;
        }
    }

    private static class RequestEventListener extends EventListener {
        @Override
        public void requestHeadersEnd(Call call, Request request) {
            // 保存最近一次请求的请求头
            Headers headers = request.headers();
            System.out.println("method="+request.method() + " : " + request.url());
            log.error("req headers=\r\n {}" , headers);
            if(StringUtils.isNotBlank(request.header("Cookie"))) {
                loginCookie += request.header("Cookie") + ";";
            }
            if(request.body() != null) {
                Buffer buffer = new Buffer();
                try {
                    request.body().writeTo(buffer);
                    String s = buffer.toString();
                    log.info("body={}", s);
                } catch (IOException e) {
                    throw new RuntimeException(e);
                }
            }
        }
        @Override
        public void responseHeadersEnd(Call call, Response response) {
            log.info("[Interceptor] Response: code=" + response.code() + " msg=" + response.message());
            log.info("***************");
            log.warn("resp header= \r\n{}", response.headers());
//            loginCookie = response.header("Cookie");
        }
    }

}
