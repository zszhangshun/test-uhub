<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <title>Vue 示例</title>
  <!-- 使用 v-cloak 避免闪烁 -->
  <style>
    [v-cloak] { display: none; }
  </style>
  
  <!-- 引入 Vue 3 (假设已经迁移) -->
  <script type="module" src="https://unpkg.com/vue@next"></script>
</head>
<body>
  <div id="app" v-cloak>{{ message }}</div>

  <script type="module">
    const app = Vue.createApp({
      data() {
        return {
          message: '你好，Vue 3！'
        };
      }
    }).mount('#app');
  </script>
</body>
</html>