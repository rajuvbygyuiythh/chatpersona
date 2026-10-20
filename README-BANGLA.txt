ChitChat Bot — দ্রুত গাইড (বাংলা)
=================================

১) জিপটা পুরোটা এক্সট্রাক্ট করুন (উইন্ডোজের ডিফল্ট Extract All ব্যবহার করুন)।
২) install.bat ডাবল ক্লিক করুন। প্রথমবার Go / Wails / WebView2 বসাতে কয়েক মিনিট লাগতে পারে।
৩) শেষে ড্যাশবোর্ড নিজে খুলবে (অথবা Y চাপুন)।
   পরে চালাতে: run-dashboard.bat
৪) ড্যাশবোর্ডের বাম দিকের মেনু: Show Mode / Background Bot।
   Message পেজে ৪টা মোড — Single Sequence, Folder Sequences, EVAChat Flow, API Chat।
৫) রিয়েল অ্যাপে না গিয়ে বটকে টেস্ট করতে: ড্যাশবোর্ড বন্ধ করে test.bat চালান।

সমস্যা হলে
-----------
- বিল্ড ফেল করলে install.bat এখন স্ক্রিনেই কম্পাইলারের শেষ লাইনগুলো দেখায়।
  পুরো লগ: source_code\build\install.log
- chitchatui.exe খুলছে না? এক্সই নিজেই একটা ছোট error dialog দেখাবে;
  একই কারণ সেভ হয়: source_code\build\bin\startup-error.log
- বিস্তারিত (বাংলা): source_code\docs\BUILD_TROUBLESHOOTING.md
- টেস্টারের গাইড: source_code\docs\LOCAL_TEST_CHAT.md

কোনো exe, data ফোল্ডার বা লগ এই জিপে দেওয়া নেই — শুধু সোর্স আর লঞ্চার।
