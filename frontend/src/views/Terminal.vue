<template>
  <div class="terminal-container">
    <div class="terminal-header">
      <h2>Agent Management</h2>
      <p>Manage and task MuliC2 agents</p>
    </div>

    <div class="agent-controls">
      <div class="control-section">
        <h3>Agent Status</h3>
        <p>Monitor and manage connected agents from this interface.</p>
      </div>
      
      <div class="control-section">
        <h3>Agent Management</h3>
        <p>Use the Agents tab in the main dashboard to view and manage connected agents.</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// Agent management functionality will be implemented here
</script>

<style scoped>
.terminal-container {
  padding: 20px;
  max-width: 1200px;
  margin: 0 auto;
}

.terminal-header {
  text-align: center;
  margin-bottom: 30px;
}

.terminal-header h2 {
  color: #2c3e50;
  margin-bottom: 10px;
}

.terminal-header p {
  color: #7f8c8d;
  font-size: 16px;
}

.agent-controls {
  display: grid;
  gap: 20px;
}

.control-section {
  background: #f8f9fa;
  padding: 20px;
  border-radius: 8px;
  border: 1px solid #e9ecef;
}

.control-section h3 {
  color: #2c3e50;
      margin-bottom: 15px;
  font-size: 18px;
    }
    
.control-section p {
  color: #495057;
        margin-bottom: 15px;
  line-height: 1.6;
}

.code-block {
  background: #2c3e50;
  color: #ecf0f1;
    padding: 15px;
  border-radius: 6px;
    font-family: 'Courier New', monospace;
  overflow-x: auto;
}

.code-block code {
  color: #ecf0f1;
}
</style>
