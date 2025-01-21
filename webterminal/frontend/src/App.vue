<template>
  <div class="container">
    <!-- 登录表单 -->
    <div v-if="!connected" class="login-form">
      <h2 class="title">SSH 终端连接</h2>
      <el-form :model="form" label-width="100px" :rules="rules" ref="formRef">
        <el-form-item label="主机" prop="host">
          <el-input v-model="form.host" placeholder="请输入主机地址"></el-input>
        </el-form-item>
        <el-form-item label="端口" prop="port">
          <el-input v-model.number="form.port" type="number" placeholder="SSH端口"></el-input>
        </el-form-item>
        <el-form-item label="用户名" prop="username">
          <el-input v-model="form.username" placeholder="SSH用户名"></el-input>
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input v-model="form.password" type="password" placeholder="SSH密码" show-password></el-input>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="submitForm" :loading="loading">
            {{ loading ? '连接中...' : '连接' }}
          </el-button>
        </el-form-item>
      </el-form>
    </div>

    <!-- 终端和AI助手 -->
    <div v-else class="workspace">
      <!-- AI助手面板 -->
      <div v-if="showAiPanel" class="ai-panel">
        <div class="ai-panel-header">
          <span class="ai-title">AI 助手</span>
          <el-switch v-model="showAiPanel" active-text="显示" inactive-text="隐藏" />
        </div>
        <div class="ai-content">
          <div class="ai-suggestion">
            <h4>推荐命令</h4>
            <div class="suggestion-main">
              <div class="command-preview">{{ aiSuggestion.command }}</div>
              <div class="command-explanation">{{ aiSuggestion.explanation }}</div>
              <el-button type="primary" size="small" @click="executeAiCommand(aiSuggestion.command)">
                执行
              </el-button>
            </div>
          </div>
          <div class="ai-alternatives">
            <h4>其他选项</h4>
            <div v-for="(cmd, index) in aiSuggestion.subCommands" :key="index" class="alternative-item">
              <span class="cmd-text">{{ cmd.cmd }}</span>
              <span class="cmd-desc">{{ cmd.desc }}</span>
              <el-button type="primary" link size="small" @click="executeAiCommand(cmd.cmd)">
                执行
              </el-button>
            </div>
          </div>
          <div class="command-history">
            <h4>最近命令</h4>
            <div class="history-list">
              <div v-for="(cmd, index) in commandHistory.slice(-5).reverse()" :key="index" class="history-item">
                {{ cmd }}
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 终端显示 -->
      <div class="terminal-wrapper">
        <div class="terminal-header">
          <span>已连接到: {{ form.username }}@{{ form.host }}</span>
          <el-button type="danger" size="small" @click="disconnect">断开连接</el-button>
        </div>
        <div ref="terminal" class="terminal-container">
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import { ElMessage } from 'element-plus'
import axios from 'axios'
import 'xterm/css/xterm.css'

const connected = ref(false)
const loading = ref(false)
const formRef = ref(null)

const form = ref({
  host: 'localhost',
  port: 22,
  username: '',
  password: ''
})

const rules = {
  host: [{ required: true, message: '请输入主机地址', trigger: 'blur' }],
  port: [{ required: true, message: '请输入端口号', trigger: 'blur' }],
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

const terminal = ref(null)
let term = null
let ws = null
const fitAddon = new FitAddon()

// AI助手相关的状态
const commandHistory = ref([])
const aiSuggestion = ref({
  command: '',
  explanation: '',
  subCommands: []
})
const showAiPanel = ref(true)

// 初始化终端
const initTerminal = () => {
  term = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: {
      background: '#1e1e1e',
      foreground: '#f0f0f0',
      cursor: '#f0f0f0',
      selection: 'rgba(255, 255, 255, 0.3)',
      black: '#000000',
      red: '#e06c75',
      green: '#98c379',
      yellow: '#d19a66',
      blue: '#61afef',
      magenta: '#c678dd',
      cyan: '#56b6c2',
      white: '#abb2bf'
    }
  })
  term.loadAddon(fitAddon)
  term.open(terminal.value)
  fitAddon.fit()

  let currentCommand = ''
  let enterPressed = false

  term.onData((data) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({
        type: 'data',
        payload: data
      }))

      // 收集命令历史
      if (data === '\r') { // 回车键
        if (currentCommand.trim()) {
          commandHistory.value.push(currentCommand.trim())
          generateAiSuggestion(currentCommand.trim())
        }
        currentCommand = ''
        enterPressed = true
      } else if (data === '\x7f') { // 退格键
        currentCommand = currentCommand.slice(0, -1)
      } else if (!enterPressed && data.length === 1 && data.charCodeAt(0) >= 32) {
        currentCommand += data
      } else {
        enterPressed = false
      }
    }
  })

  window.addEventListener('resize', () => {
    fitAddon.fit()
    if (ws && ws.readyState === WebSocket.OPEN) {
      const { rows, cols } = term
      ws.send(JSON.stringify({
        type: 'resize',
        payload: {
          width: cols,
          height: rows
        }
      }))
    }
  })
}

// AI建议生成函数
const generateAiSuggestion = (lastCommand) => {
  if (lastCommand.startsWith('cd ')) {
    aiSuggestion.value = {
      command: 'ls -la',
      explanation: '列出新目录的详细内容',
      subCommands: [
        { cmd: 'ls', desc: '仅列出文件名' },
        { cmd: 'ls -l', desc: '显示详细信息' },
        { cmd: 'ls -la', desc: '显示所有文件的详细信息' }
      ]
    }
  } else if (lastCommand.startsWith('git clone ')) {
    aiSuggestion.value = {
      command: 'cd ' + lastCommand.split(' ').pop().split('/').pop().replace('.git', ''),
      explanation: '进入克隆的项目目录',
      subCommands: [
        { cmd: 'cd ' + lastCommand.split(' ').pop().split('/').pop().replace('.git', ''), desc: '进入项目目录' },
        { cmd: 'ls -la', desc: '查看项目文件' },
        { cmd: 'git branch', desc: '查看分支信息' }
      ]
    }
  } else {
    aiSuggestion.value = {
      command: 'history',
      explanation: '查看命令历史',
      subCommands: [
        { cmd: 'clear', desc: '清屏' },
        { cmd: 'pwd', desc: '显示当前路径' },
        { cmd: 'ls', desc: '列出文件' }
      ]
    }
  }
}

// 执行AI建议的命令
const executeAiCommand = (command) => {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({
      type: 'data',
      payload: command + '\r'
    }))
  }
}

// 连接到后端
const submitForm = async () => {
  if (!formRef.value) return
  
  try {
    await formRef.value.validate()
    loading.value = true
    
    const response = await axios.post('http://localhost:8080/api/terminal', form.value)
    const { session_id } = response.data

    const wsUrl = `ws://localhost:8080/ws/terminal/${session_id}?host=${form.value.host}&username=${form.value.username}&password=${form.value.password}`
    ws = new WebSocket(wsUrl)

    ws.onopen = () => {
      connected.value = true
      loading.value = false
      ElMessage.success('连接成功')
      setTimeout(() => {
        initTerminal()
      }, 100)
    }

    ws.onmessage = (event) => {
      const msg = JSON.parse(event.data)
      if (msg.type === 'data') {
        term.write(msg.payload)
      }
    }

    ws.onclose = () => {
      disconnect()
      ElMessage.warning('连接已断开')
    }

    ws.onerror = (error) => {
      console.error('WebSocket error:', error)
      disconnect()
      ElMessage.error('连接出错')
    }
  } catch (error) {
    loading.value = false
    console.error('Connection error:', error)
    ElMessage.error(error.response?.data?.message || '连接失败')
  }
}

const disconnect = () => {
  if (ws) {
    ws.close()
  }
  if (term) {
    term.dispose()
    term = null
  }
  connected.value = false
  loading.value = false
}
</script>

<style scoped>
.container {
  height: 100vh;
  padding: 20px;
  box-sizing: border-box;
  background-color: #f5f7fa;
}

.workspace {
  display: flex;
  gap: 20px;
  height: 100%;
}

.ai-panel {
  width: 300px;
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
  display: flex;
  flex-direction: column;
  height: 100%;
}

.ai-panel-header {
  padding: 12px 16px;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.ai-title {
  font-size: 16px;
  font-weight: 500;
  color: #303133;
}

.ai-content {
  flex: 1;
  padding: 16px;
  overflow-y: auto;
}

.ai-suggestion, .ai-alternatives, .command-history {
  margin-bottom: 20px;
}

.suggestion-main {
  background: #f5f7fa;
  border-radius: 4px;
  padding: 12px;
  margin: 8px 0;
}

.command-preview {
  font-family: monospace;
  font-size: 14px;
  color: #409eff;
  margin-bottom: 8px;
  word-break: break-all;
}

.command-explanation {
  font-size: 12px;
  color: #606266;
  margin-bottom: 8px;
}

.alternative-item {
  display: flex;
  align-items: center;
  padding: 8px;
  border-bottom: 1px solid #ebeef5;
}

.cmd-text {
  font-family: monospace;
  font-size: 12px;
  color: #606266;
  margin-right: 8px;
  flex: 1;
  word-break: break-all;
}

.cmd-desc {
  font-size: 12px;
  color: #909399;
  margin-right: 8px;
  flex: 1;
}

.history-item {
  font-family: monospace;
  font-size: 12px;
  color: #606266;
  padding: 4px 8px;
  background: #f5f7fa;
  margin-bottom: 4px;
  border-radius: 4px;
  word-break: break-all;
}

.terminal-wrapper {
  flex: 1;
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  height: 100%;
}

.title {
  text-align: left;
  margin-bottom: 20px;
  color: #303133;
  font-size: 20px;
}

.login-form {
  width: 360px;
  height: fit-content;
  padding: 20px;
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
}

.terminal-header {
  padding: 8px 16px;
  background: #f5f7fa;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 40px;
}

.terminal-container {
  flex: 1;
  background: #1e1e1e;
  padding: 0;
  overflow: hidden;
  display: flex;
  align-items: flex-start;
  justify-content: flex-start;
  position: relative;
}

:deep(.xterm) {
  padding: 4px;
  height: 100%;
  width: 100%;
}

:deep(.xterm-viewport) {
  overflow-y: auto !important;
}

:deep(.xterm-screen) {
  text-align: left;
}
</style>
