# pingtunnel با حالت Echo اختیاری

این ریپو اکنون نسخهٔ تغییر‌یافتهٔ [esrrhs/pingtunnel](https://github.com/esrrhs/pingtunnel) است.
کد WaterWall از شاخهٔ اصلی حذف شده و در تاریخچهٔ Git قابل بازیابی می‌ماند.

حالت `echo_mode` دادهٔ رمز‌شده را در هر دو جهت با Echo Request منتقل می‌کند و
Echo Reply با payload دقیقاً یکسان می‌فرستد. باید در **هر دو سرور** فعال باشد.
حالت قدیمی پیش‌فرض باقی مانده است. این تغییر تضمین عبور از DPI نیست.

## دانلود برای Linux x86_64 / amd64

```sh
curl -fL https://raw.githubusercontent.com/moeinshahcheraghi/WaterWall/main/pingtunnel -o pingtunnel
curl -fL https://raw.githubusercontent.com/moeinshahcheraghi/WaterWall/main/SHA256SUMS -o SHA256SUMS
sha256sum --ignore-missing -c SHA256SUMS
chmod +x pingtunnel
./pingtunnel -version
```

یا [ZIP باینری، کانفیگ‌ها و راهنما](https://github.com/moeinshahcheraghi/WaterWall/raw/refs/heads/main/downloads/pingtunnel-echo-linux-amd64.zip)
را دانلود کن. باینری آماده برای Linux amd64 است؛ معماری دیگر باید از سورس ساخته شود.

## تنظیم OpenVPN TCP/2083

روی سرور **خارج**:

```sh
curl -fL https://raw.githubusercontent.com/moeinshahcheraghi/WaterWall/main/config-server-echo-sample.json -o server.json
chmod 600 server.json
```

روی سرور **ایران**:

```sh
curl -fL https://raw.githubusercontent.com/moeinshahcheraghi/WaterWall/main/config-client-echo-sample.json -o client.json
chmod 600 client.json
```

در هر دو فایل، `password` را با رمز قوی و یکسان جایگزین کن. در `client.json`
آی‌پی نمونهٔ `2.2.2.2` را با آی‌پی واقعی خارج عوض کن.

OpenVPN خارج باید از `127.0.0.1:2083` قابل دسترسی باشد. پورت `2083` ایران باید
آزاد و برای کاربران باز باشد. در فایروال هر دو سرور، ICMP Echo Request از سرور
مقابل مجاز باشد؛ پذیرش فقط درخواست از ایران و پاسخ از خارج کافی نیست.

لینوکس ممکن است خودش هم در کنار برنامه Echo Reply بفرستد. روی میزبان یا
namespace اختصاصی ابتدا مقدار فعلی را یادداشت کن:

```sh
sysctl net.ipv4.icmp_echo_ignore_all
```

سپس در صورت مناسب بودن برای همان میزبان اختصاصی:

```sh
sudo sysctl -w net.ipv4.icmp_echo_ignore_all=1
```

این تنظیم پاسخ ping عادی میزبان را هم متوقف می‌کند. برنامه آن را خودکار تغییر
نمی‌دهد؛ پس از کنار گذاشتن تونل مقدار قبلی را برگردان.

اجرای خارج:

```sh
sudo ./pingtunnel -c server.json
```

اجرای ایران:

```sh
sudo ./pingtunnel -c client.json
```

در کلاینت OpenVPN، گواهی‌ها و کلیدها را حفظ و آدرس اتصال را به آی‌پی واقعی ایران عوض کن:

```conf
proto tcp-client
remote 1.1.1.1 2083
```

`1.1.1.1` فقط مثال است و باید با آی‌پی واقعی ایران جایگزین شود.

## ساخت و اعتبارسنجی

با نسخهٔ Go مشخص‌شده در `go.mod`:

```sh
go build -mod=readonly -trimpath -o pingtunnel-echo ./cmd
go test ./... -skip '^TestE2E' -count=1
```

تست‌های واحد، race و انتقال TCP/UDP/SOCKS با ICMP واقعی در namespace جدا پاس شده‌اند.
مقاومت مقابل DPI اپراتور و اتصال به سرورهای واقعی کاربر آزمایش نشده است.
نوع ICMP، اندازه و نرخ ترافیک همچنان قابل مشاهده‌اند.

- [جزئیات حالت Echo و تست‌ها](ECHO_MODE.md)
- [مستندات عمومی upstream](README.upstream.md)
- [راهنمای تنظیمات](USAGE.md)
- [مجوز upstream](LICENSE)
# ping
# ping
# ping
