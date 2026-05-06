# 🚀 دليل النشر على Cloudflare

هذا الدليل يشرح خطوة بخطوة كيفية نشر تطبيق Foodie على Cloudflare Pages و Workers.

---

## 📋 متطلبات النشر

1. حساب Cloudflare (مجاني)
2. Node.js 18+ و npm
3. Wrangler CLI
4. GitHub repository (لـ CI/CD)

---

## 🌐 الخطوة 1: إعداد Cloudflare

### 1.1 إنشاء حساب Cloudflare

1. اذهب إلى [cloudflare.com](https://dash.cloudflare.com/sign-up)
2. سجل حساب جديد ببريدك الإلكتروني
3. أكد البريد الإلكتروني

### 1.2 الحصول على API Token

1. في لوحة التحكم، اذهب إلى **My Profile** → **API Tokens**
2. اضغط **Create Token**
3. اختر **Custom Token**
4. املأ التفاصيل:
   - **Token name**: `Foodie Deployment`
   - **Permissions**:
     - `Cloudflare Pages:Edit`
     - `Account Settings:Read`
     - `Workers Scripts:Edit`
     - `Zone Settings:Read`
5. اضغط **Continue to Summary** ثم **Create Token**
6. **انسخ التوكن واحفظه في مكان آمن** ⚠️

### 1.3 الحصول على Account ID

1. في لوحة التحكم، انظر إلى الجانب الأيمن
2. ستجد **Account ID** - انسخه

---

## 🛠️ الخطوة 2: إعداد Wrangler CLI

### 2.1 تثبيت Wrangler

```bash
# عالمياً
npm install -g wrangler

# أو محلياً في المشروع
npm install --save-dev wrangler
```

### 2.2 تسجيل الدخول

```bash
wrangler login
```

سيفتح المتصفح تلقائياً للمصادقة.

---

## 📦 الخطوة 3: نشر Frontend على Pages

### 3.1 إنشاء مشاريع Pages

```bash
# Customer App
cd frontend/customer
wrangler pages project create foodie-customer

# Restaurant App  
cd ../restaurant
wrangler pages project create foodie-restaurant

# Driver App
cd ../driver
wrangler pages project create foodie-driver

# Admin App
cd ../admin
wrangler pages project create foodie-admin
```

### 3.2 النشر اليدوي

```bash
# Customer App
cd frontend/customer
wrangler pages deploy . --project-name=foodie-customer

# سيكون الرابط: https://foodie-customer.pages.dev
```

كرر نفس الخطوات للتطبيقات الأخرى.

### 3.3 إعداد Custom Domain (اختياري)

1. في Cloudflare Dashboard، اذهب إلى **Pages**
2. اختر المشروع
3. اذهب إلى **Custom Domains**
4. أضف نطاقك الخاص (مثل: `app.yourdomain.com`)

---

## ⚙️ الخطوة 4: نشر Backend على Workers

### 4.1 تعديل الكود للـ Workers

أنشئ ملف `backend/cmd/server/worker.go`:

```go
package main

import (
    "github.com/syumai/workers"
)

func main() {
    workers.Serve(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Your handler logic
    }))
}
```

### 4.2 إعداد Wrangler.toml

```toml
name = "foodie-api"
main = "backend/cmd/server/worker.go"
compatibility_date = "2024-01-01"

[env.production]
vars = { APP_ENV = "production" }

# Secrets (ستُضاف عبر CLI)
# JWT_SECRET
# DATABASE_URL  
# REDIS_URL
```

### 4.3 بناء ونشر

```bash
# بناء للـ WASM
GOOS=js GOARCH=wasm go build -o main.wasm ./cmd/server

# نشر
wrangler deploy

# إضافة Secrets
echo "your-jwt-secret" | wrangler secret put JWT_SECRET
echo "postgres://..." | wrangler secret put DATABASE_URL
echo "redis://..." | wrangler secret put REDIS_URL
```

---

## 🔧 الخطوة 5: إعداد GitHub Actions (CI/CD)

### 5.1 إضافة Secrets على GitHub

1. في repository على GitHub، اذهب إلى **Settings** → **Secrets and variables** → **Actions**
2. أضف الأسرار التالية:

```
CLOUDFLARE_API_TOKEN=your-api-token-here
CLOUDFLARE_ACCOUNT_ID=your-account-id-here
JWT_SECRET=your-jwt-secret
DATABASE_URL=your-database-url
REDIS_URL=your-redis-url
STRIPE_SECRET_KEY=your-stripe-key
```

### 5.2 إنشاء Workflows

تم إنشاء الملفات تلقائياً في `.github/workflows/`:

- `ci.yml` - اختبارات CI
- `deploy-cloudflare.yml` - نشر تلقائي

### 5.3 تشغيل CI/CD

كل push إلى `main` سيتم نشره تلقائياً!

---

## 🗄️ الخطوة 6: إعداد قاعدة البيانات

### 6.1 PostgreSQL على Supabase (مجاني)

1. اذهب إلى [supabase.com](https://supabase.com)
2. أنشئ مشروع جديد
3. في **Database** → **Connection string**، انسخ الرابط
4. أضفه كـ `DATABASE_URL` secret

### 6.2 Redis على Upstash (مجاني)

1. اذهب إلى [upstash.com](https://upstash.com)
2. أنشئ قاعدة بيانات Redis
3. انسخ **REST URL**
4. أضفه كـ `REDIS_URL` secret

### 6.3 تهيئة الجداول

```bash
# على Supabase SQL Editor، شغل:
cat scripts/init.sql | psql $DATABASE_URL
```

---

## 🌐 الخطوة 7: إعداد النطاق المخصص

### 7.1 إضافة النطاق إلى Cloudflare

1. في Dashboard، اذهب إلى **Websites**
2. اضغط **Add Site**
3. أدخل نطاقك (مثال: `foodie.app`)
4. اتبع خطوات التحقق

### 7.2 إعداد DNS Records

| Type | Name | Content | Proxy Status |
|------|------|---------|--------------|
| CNAME | @ | foodie-customer.pages.dev | Proxied |
| CNAME | restaurant | foodie-restaurant.pages.dev | Proxied |
| CNAME | driver | foodie-driver.pages.dev | Proxied |
| CNAME | admin | foodie-admin.pages.dev | Proxied |
| CNAME | api | foodie-api.your-account.workers.dev | Proxied |

### 7.3 إعداد SSL/TLS

1. اذهب إلى **SSL/TLS**
2. اختر **Full (strict)**
3. فعل **Always Use HTTPS**

---

## 🔒 الخطوة 8: إعداد الأمان

### 8.1 Workers Security

```toml
# wrangler.toml
[env.production]
routes = [
  { pattern = "api.foodie.app/*", zone_name = "foodie.app" }
]

# Rate Limiting
[[unsafe.bindings]]
name = "RATE_LIMITER"
type = "ratelimit"
namespace_id = "1001"

# WAF Rules
[env.production.waf]
mode = "on"
```

### 8.2 CORS Configuration

```go
// في backend
app.Use(cors.New(cors.Config{
    AllowOrigins: "https://foodie.app, https://*.foodie.app",
    AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
    AllowHeaders: "Origin,Content-Type,Accept,Authorization",
}))
```

---

## 📊 الخطوة 9: المراقبة والتحليلات

### 9.1 Cloudflare Analytics

- تلقائياً متاح في Dashboard
- مراقبة الطلبات، الأداء، الأخطاء

### 9.2 إضافة Custom Analytics

```javascript
// في frontend
fetch('/api/v1/analytics/event', {
    method: 'POST',
    body: JSON.stringify({
        event: 'page_view',
        page: window.location.pathname
    })
});
```

---

## 🧪 الخطوة 10: اختبار النشر

### 10.1 فحص النقاط المهمة

```bash
# 1. Health Check
curl https://api.foodie.app/api/v1/health

# 2. Test Auth
curl -X POST https://api.foodie.app/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password"}'

# 3. Test Frontend
curl https://foodie.app/
curl https://restaurant.foodie.app/
```

### 10.2 Lighthouse CI

```bash
npm install -g @lhci/cli
lhci autorun
```

---

## 🔄 التحديثات المستقبلية

### تحديث يدوي

```bash
# Frontend
cd frontend/customer
wrangler pages deploy . --project-name=foodie-customer

# Backend
wrangler deploy
```

### تحديث تلقائي (GitHub Actions)

كل push إلى `main` ينشر تلقائياً!

---

## 🆘 استكشاف الأخطاء

### مشكلة: Build Fails

```bash
# تحقق من Go version
go version  # يجب أن يكون 1.23+

# تنظيف cache
go clean -cache
go mod tidy
```

### مشكلة: Secrets Not Found

```bash
# التحقق من Secrets
wrangler secret list

# إعادة إضافة Secret
echo "value" | wrangler secret put SECRET_NAME
```

### مشكلة: Database Connection Failed

```bash
# التحقق من Connection String
wrangler secret get DATABASE_URL

# اختبار الاتصال
psql $DATABASE_URL -c "SELECT 1"
```

### مشكلة: CORS Errors

تأكد من إعدادات CORS في Backend وإضافة النطاق إلى القائمة المسموح بها.

---

## 💡 نصائح الإنتاج

### الأداء
- فعل **Auto Minify** في Cloudflare
- فعل **Brotli Compression**
- استخدم **Cloudflare Images** للصور

### الأمان
- فعل **Security Level: High**
- فعل **Bot Fight Mode**
- استخدم **Cloudflare Access** للـ Admin Panel

### التكلفة
- الحد المجاني: 100,000 طلب/يوم
- قاعدة البيانات: Supabase Free Tier (500MB)
- Redis: Upstash Free Tier (10,000 commands/يوم)

---

## 📞 دعم

- **Cloudflare Docs**: https://developers.cloudflare.com
- **Discord Community**: https://discord.gg/cloudflare
- **GitHub Issues**: https://github.com/yourusername/foodie/issues

---

## ✅ قائمة المراجعة

قبل إطلاق الموقع:

- [ ] API Token جاهز
- [ ] Account ID محدد
- [ ] Projects created على Pages
- [ ] Backend deployed على Workers
- [ ] Database متصلة
- [ ] Secrets مضافة
- [ ] Custom Domain مضبوط
- [ ] SSL/TLS مفعل
- [ ] Tests ناجحة
- [ ] GitHub Actions تعمل

---

**🎉 مبروك! تطبيقك الآن live على Cloudflare!**
