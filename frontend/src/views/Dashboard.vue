<template>
  <div class="dashboard">
    <h1>🧪 Dashboard Test Component</h1>
    <p>If you can see this, the Dashboard component is loading correctly!</p>
    <p>Timestamp: {{ new Date().toLocaleString() }}</p>
    
    <div style="background: #e8f5e8; padding: 15px; margin: 15px 0; border-radius: 8px; border: 2px solid #67c23a;">
      <h3>✅ Dashboard Component Successfully Loaded!</h3>
      <p>This confirms the component is rendering and the router is working.</p>
    </div>
    
    <div style="background: #fff3cd; padding: 15px; margin: 15px 0; border-radius: 8px; border: 2px solid #ffc107;">
      <h3>🔍 Next Steps</h3>
      <p>Since this component is loading, the issue was likely in the complex template or script logic.</p>
      <p>We can now gradually restore the functionality.</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'

onMounted(() => {
  console.log('🚀 Dashboard test component mounted successfully!')
})
</script>

<style scoped>
.dashboard {
  padding: 20px;
  max-width: 1200px;
  margin: 0 auto;
}

h1 {
  color: #409eff;
  text-align: center;
  margin-bottom: 20px;
}

p {
  font-size: 16px;
  line-height: 1.6;
  margin-bottom: 10px;
}
</style>
