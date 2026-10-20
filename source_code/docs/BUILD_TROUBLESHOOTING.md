# বিল্ড / ড্যাশবোর্ড সমস্যা — দ্রুত সমাধান

## ১. `install.bat` চালালাম, কিন্তু `chitchatui.exe` তৈরি হয়নি

`install.bat` এখন স্ক্রিনেই দেখায় সমস্যাটা কোথায়:

1. **`Dashboard frontend files are missing`** — জিপটা অসম্পূর্ণ এক্সট্রাক্ট হয়েছে
   (অ্যান্টিভাইরাস ফাইল আটকে দিতে পারে)। পুরো জিপটা নতুন ফোল্ডারে আবার এক্সট্রাক্ট
   করে `install.bat` চালান।
2. **`wails build` ব্যর্থ** — স্ক্রিপ্ট নিজেই `build\install.log`-এর শেষ ২৫ লাইন
   দেখায়, তারপর `go build` ফলব্যাক চালায়। ফলের লাইনটা পাঠালেই সমস্যা বোঝা যায়।
3. **`go: cannot find main module` / `missing go.sum entry`** — ইন্টারনেট/Cookie
   সমস্যা। `go env GOPROXY` ঠিক আছে কিনা দেখুন, তারপর আবার চালান।

সব লগ থাকে: `source_code\build\install.log`।

## ২. `chitchatui.exe` আছে, কিন্তু উইন্ডো খোলে না

এক্সই এখন **নিজেই কারণ দেখায়**:

- একটি ছোট Windows error dialog আসবে ("ChitChat Bot could not start")।
- একই লেখা সেভ হয় `build\bin\startup-error.log`-এ।
- `run-dashboard.bat` ৬ সেকেন্ড অপেক্ষা করে; এক্সই বন্ধ হয়ে গেলে টার্মিনালেই
  `startup-error.log`-এর কারণ দেখিয়ে দেয়।

সাধারণ কারণ:

| কারণ | সমাধান |
|---|---|
| ডেটা ফোল্ডার অন্য ইনস্ট্যান্সে খোলা (badger lock) | সব `chitchatui.exe` এবং `test.bat` বন্ধ করে আবার চালান |
| `data\` ফোল্ডারে লেখার অনুমতি নেই | প্রজেক্টটা `C:\` বা Desktop-এর সাধারণ ফোল্ডারে রাখুন, Program Files-এ নয় |
| WebView2 নেই | https://developer.microsoft.com/microsoft-edge/webview2/ থেকে ইনস্টল করুন |
| অ্যান্টিভাইরাস/SmartScreen এক্সই আটকে দিয়েছে | এক্সই-টা Allow/Unblock করুন; `startup-error.log` তৈরি হয়নি মানে এটাই সবচেয়ে সম্ভাব্য কারণ |

## ৩. ড্যাশবোর্ড খোলে কিন্তু সাদা/ফাঁকা

- `cmd\chitchatui\frontend\web\index.html` ফাইলটা জিপে আছে কিনা দেখুন
  (আগের নাম `frontend\dist`; এখন `frontend\web`)।
- `install.bat build` দিয়ে আবার বিল্ড করুন।

## ৪. `test.bat` চালাতে চাই

- আগে ড্যাশবোর্ড বন্ধ করুন (একই ডেটা ফোল্ডার)।
- `test.bat` → ৪টা মোডের মেনু → চ্যাট। বিস্তারিত: `docs/LOCAL_TEST_CHAT.md`।

## ৫. ফাইলগুলো কোথায়

```
install.bat                   → সব ধাপ (Go, Wails, WebView2, build)
run-dashboard.bat             → শুধু এক্সই চালু (ব্যর্থ হলে কারণ দেখায়)
test.bat                      → অফলাইন চ্যাট টেস্টার (৪ মোড)
build\install.log             → বিল্ড লগ
build\bin\chitchatui.exe      → ড্যাশবোর্ড
build\bin\startup-error.log   → এক্সই না খুললে কারণ
```
