# بیلد و اجرای pingtunnel

این بسته فقط سورس است. Go 1.27.3 را نصب کن و از ریشهٔ پروژه اجرا کن:

```bash
go mod download
go mod verify
bash scripts/build-linux.sh
```

خروجی هر سه معماری در `dist/` ساخته می‌شود. هر باینری هر دو نقش ایران/client و خارج/server را دارد.

مسیر ارتباط:

```text
کاربر OpenVPN → Iran:2083/TCP → ICMP → Foreign → OpenVPN TCP
                        client             server
```

server یعنی سرور خارج و client یعنی سرور ایران، نه دستگاه کاربر.
این نسخه تضمین نامرئی‌شدن در برابر DPI یا ظرفیت قطعی ندارد. سناریوی هدف حدود
۱۰۰Mbps/۳۰۰ کاربر در هر سرور ایران و با تقسیم برابر ۲۵۰Mbps/۷۵۰ کاربر در هر سرور خارج است.

## ۱. انتخاب باینری آماده یا بیلد

روی هر سرور `uname -m` را اجرا کن:

| خروجی | باینری در `dist/` |
|---|---|
| x86_64 | pingtunnel-linux-amd64 |
| aarch64 / arm64 | pingtunnel-linux-arm64 |
| armv7l | pingtunnel-linux-armv7 |

ARMv7 نسخهٔ ۳۲بیتی است. ARM64 برای بیشتر سرورهای ARM جدید مناسب است.
بیلد هر سه معماری انجام شده؛ اجرای بومی فقط روی amd64 این محیط بررسی شده است.

برای باینری آماده از ریشهٔ بسته:

```bash
(cd dist && sha256sum -c SHA256SUMS)
# برای ARM نام فایل را مطابق جدول عوض کن.
sudo install -m 0755 dist/pingtunnel-linux-amd64 /usr/local/bin/pingtunnel
/usr/local/bin/pingtunnel -version
```

برای ساخت از سورس، Go 1.27.3 مطابق `go.mod` را از https://go.dev/dl/ نصب کن.
آرشیو Go باید با معماری میزبان بیلد سازگار باشد؛ checksum رسمی دانلود را بررسی کن.
سپس از ریشهٔ مخزن:

```bash
go version
go mod download
go mod verify
bash scripts/build-linux.sh
```

اسکریپت هر سه معماری را cross-compile می‌کند؛ اجرای باینری ARM روی میزبان x86 ممکن نیست.
بیلد فقط برای معماری همان میزبان نیز ممکن است:

```bash
CGO_ENABLED=0 go build -buildvcs=false -mod=readonly -trimpath -o pingtunnel ./cmd
```

## ۲. تنظیم و اجرای سرور خارج

OpenVPN باید از قبل با TCP فعال باشد. اگر روی همین سرور و پورت ۲۰۸۳ گوش می‌دهد،
مقدار target سمت ایران `127.0.0.1:2083` است؛ این آدرس از سمت خارج تفسیر می‌شود.
اگر OpenVPN روی پورت یا میزبان دیگری است، target را مطابق همان تغییر بده.

```bash
sudo install -d -m 0700 /etc/pingtunnel
sudo install -m 0600 config-server-capacity-sample.json /etc/pingtunnel/server.json
sudo nano /etc/pingtunnel/server.json
```

`password` نمونه را با یک رمز قوی و مشترک با ایران عوض کن. برای تولید رمز می‌توانی
`openssl rand -hex 32` اجرا کنی. رمز واقعی را در Git، README یا پیام عمومی قرار نده.
`encrypt: chacha20` و `echo_mode: true` را در هر دو سمت یکسان نگه دار.

اجرای آزمایشی در foreground:

```bash
sudo /usr/local/bin/pingtunnel -c /etc/pingtunnel/server.json
```

## ۳. تنظیم و اجرای سرور ایران

```bash
sudo install -d -m 0700 /etc/pingtunnel
sudo install -m 0600 config-client-capacity-sample.json /etc/pingtunnel/client.json
sudo nano /etc/pingtunnel/client.json
```

این موارد را تنظیم کن:

- `server`: آدرس IPv4 سرور خارج، بدون پورت؛ آدرس نمونه `2.2.2.2` را عوض کن.
- `listen`: مقدار `0.0.0.0:2083` برای اتصال کاربران به ایران.
- `target`: آدرس و پورت OpenVPN از دید سرور خارج، مثلاً `127.0.0.1:2083`.
- `password`: دقیقاً همان رمز خارج.
- `tcp_buffer_bytes: 262144` و `tcp_window_frames: 512`: نقطهٔ شروع پروفایل ظرفیت.

```bash
sudo /usr/local/bin/pingtunnel -c /etc/pingtunnel/client.json
```

پورت TCP/2083 ایران باید آزاد باشد. کلاینت OpenVPN کاربر به IP ایران و TCP/2083 وصل می‌شود.
خود تونل بین دو سرور ICMP است و پورت TCP/8000 نقشی در اجرای آن ندارد.
فایروال میزبان و ارائه‌دهنده باید ICMP Echo Request/Reply بین دو سمت و TCP/2083 ورودی ایران
را اجازه دهد. قواعد موجود را بدون بررسی پاک نکن.

برنامه برای raw ICMP به دسترسی root یا CAP_NET_RAW نیاز دارد. دستورات بالا با sudo هستند.
مقدار معمول `net.ipv4.icmp_echo_ignore_all=0` را صرفاً به خاطر این تونل به ۱ تغییر نده؛
کد از پاسخ کرنل استفاده می‌کند و پاسخ تکراری نمی‌فرستد. پس از تغییر سیاست Echo برنامه را restart کن.

## ۴. اجرای دائم با systemd

پس از موفقیت اجرای دستی، با Ctrl+C آن را ببند. روی هر سرور فایل زیر را در
`/etc/systemd/system/pingtunnel.service` بساز. نمونه برای خارج است؛ روی ایران فقط
`server.json` را در ExecStart به `client.json` تغییر بده.

```ini
[Unit]
Description=Ping ICMP tunnel
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/pingtunnel -c /etc/pingtunnel/server.json
Restart=on-failure
RestartSec=3
LimitNOFILE=65536
Environment=GOMAXPROCS=4

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now pingtunnel
sudo systemctl status pingtunnel --no-pager
sudo journalctl -u pingtunnel -n 80 --no-pager
```

ابتدا خارج و سپس ایران را راه‌اندازی کن. روی سرورهای با CPU متفاوت GOMAXPROCS را متناسب تنظیم کن.
برای ارتقا سرویس را متوقف، باینری را جایگزین و دوباره start کن؛ از نسخهٔ قبلی برای rollback نسخه بگیر.

## ۵. بررسی عملکرد

اتصال OpenVPN، دانلود و تأخیر کاربر را ابتدا بدون بار و سپس با ۵۰ و ۱۰۰Mbps روی یک جفت سرور
بسنج. CPU و RAM هر دو سمت، retransmission و تأخیر p95/p99 را ثبت کن. ping کوچک به‌تنهایی
شاخص سرعت تونل نیست. راهنمای `DIAGNOSTICS.fa.md` را بخوان.

## ۶. انتقال سورس به مخزن Git خودت

ZIP پوشهٔ `.git` و رمز واقعی ندارد. محتویات پوشهٔ `ping/` داخل ZIP را روی checkout مخزن خودت
قرار بده؛ پوشهٔ `.git` مخزن خودت را نگه دار. کپی روی نسخهٔ قدیمی فایل‌های اضافی را حذف نمی‌کند؛
برای جایگزینی تمیز ابتدا از checkout قبلی پشتیبان بگیر و فایل‌های قدیمی پروژه را کنار بگذار.
سپس تغییرها را بررسی و خودت push کن:

```bash
git status --short
git diff --stat
git add -A
git diff --cached --stat
git commit -m "Improve ICMP tunnel performance and document Linux deployment"
git push
```

قبل از commit بررسی کن هیچ کانفیگ واقعی یا کلیدی stage نشده باشد. `dist/` در Git نادیده گرفته
می‌شود؛ در صورت نیاز باینری‌های ساخته‌شده را به GitHub Release اضافه کن.
این بسته به‌تنهایی مخزن Git یا تاریخچهٔ commit نیست، ولی تمام فایل‌های سورس موردنیاز را دارد.

