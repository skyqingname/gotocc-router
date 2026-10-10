<template>
  <div class="preview-shell">
    <aside class="preview-sidebar">
      <div class="preview-brand"><span class="brand-mark">G</span><div>GoToCC<span>AI ROUTER</span></div></div>
      <p class="nav-caption">控制台</p>
      <div class="nav-item"><Icon name="chartBar" size="sm" />仪表盘</div>
      <div class="nav-item active"><Icon name="key" size="sm" />API 密钥</div>
      <div class="nav-item"><Icon name="chartBar" size="sm" />使用记录</div>
      <p class="nav-caption second-caption">管理</p>
      <button class="nav-item nav-button" @click="showAdmin = true"><Icon name="arrowsUpDown" size="sm" />智能路由设置</button>
      <div class="sidebar-note"><span class="demo-dot" />交互预览<br><small>仅使用演示数据</small></div>
    </aside>
    <main class="preview-main">
      <header class="preview-topbar"><span>API 密钥 <span class="crumb">/ 编辑密钥</span></span><span class="preview-pill">功能预览</span></header>
      <div class="preview-content">
        <div class="page-heading"><div><p class="eyebrow">SMART ROUTING</p><h1>让每个密钥，按你的顺序路由。</h1><p>用户自主排序，管理员统一控制权限。</p></div><button class="btn btn-secondary" data-test="open-admin" @click="showAdmin = true"><Icon name="cog" size="sm" class="mr-2" />管理员设置</button></div>
        <div class="preview-grid">
          <section class="edit-card">
            <div class="edit-card-header"><h2>编辑密钥</h2><span class="key-caption">API KEY · 101</span></div>
            <div class="edit-card-body">
              <label class="input-label" for="preview-name">名称</label><input id="preview-name" v-model="name" class="input" />
              <label class="input-label mode-label">路由模式</label>
              <div class="mode-tabs"><span>固定分组</span><span class="selected">智能路由</span></div>
              <p class="mode-hint">每个请求由服务端自动选择一个有权限的分组。</p>
              <RoutingPriorityPanel :key="revision" ref="panel" scope="personal" :key-id="101" :disabled="saving" />
            </div>
            <div class="edit-card-footer"><p role="status">{{ message }}</p><button class="btn btn-secondary" @click="cancel">取消</button><button class="btn btn-primary" data-test="save-user" :disabled="saving" @click="save">{{ saving ? '保存中…' : '更新' }}</button></div>
          </section>
          <aside class="preview-guide">
            <div class="guide-status"><span class="status-indicator" :class="{ closed: !adminPolicy.allow_user_override }" />{{ adminPolicy.allow_user_override ? '管理员已允许自定义' : '管理员已关闭自定义' }}</div>
            <h2>试一试新的设置</h2>
            <ol class="guide-steps"><li><b>01</b><div><strong>调整优先级</strong><p>点击“自定义优先级”，用上下箭头调整分组顺序。</p></div></li><li><b>02</b><div><strong>按模型单独设置</strong><p>添加模型规则，让不同模型使用不同的首选分组。</p></div></li><li><b>03</b><div><strong>体验管理员开关</strong><p>打开管理员设置并关闭权限，保存后查看只读状态。</p></div></li></ol>
            <div class="guide-note"><Icon name="shield" size="md" /><p>关闭权限后采用管理员顺序。你的自定义配置会保留，重新开启后恢复。</p></div>
            <div class="saved-order"><p class="eyebrow">当前已保存</p><h3>{{ adminPolicy.allow_user_override && preference ? '此密钥的自定义顺序' : '管理员默认顺序' }}</h3><div v-for="(group,index) in effectiveGroups" :key="group.id"><span>{{ index + 1 }}</span>{{ group.name }}</div></div>
          </aside>
        </div>
      </div>
    </main>
    <AutoRoutingPolicyModal :show="showAdmin" @close="showAdmin = false" @saved="adminSaved" />
  </div>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import RoutingPriorityPanel from '@/components/keys/RoutingPriorityPanel.vue'
import AutoRoutingPolicyModal from '@/components/admin/group/AutoRoutingPolicyModal.vue'
import { adminPolicy, preference, demoGroups, revision } from './fixtures'
const panel = ref<InstanceType<typeof RoutingPriorityPanel> | null>(null)
const showAdmin = ref(false)
const saving = ref(false)
const name = ref('智能')
const message = ref('')
const effectiveGroups = computed(() => {
  const ids = adminPolicy.value.allow_user_override && preference.value ? [...preference.value.default_group_order,...adminPolicy.value.default_group_order] : adminPolicy.value.default_group_order
  return [...new Set([...ids,...demoGroups.map(group=>group.id)])].map(id=>demoGroups.find(group=>group.id===id)!).filter(Boolean)
})
function cancel() { revision.value++; message.value = '已取消未保存的更改' }
function adminSaved() { message.value = '管理员设置已保存'; revision.value++ }
async function save() {
  if (panel.value?.validate() === false) return
  saving.value = true
  try { await panel.value?.save(101); message.value = '已保存，当前密钥按新顺序路由' }
  catch { message.value = '保存失败，请重试' }
  finally { saving.value = false }
}
</script>
<style>
.preview-shell{min-height:100vh;background:#151515;color:#e9e9ed;font-family:Inter,"Microsoft YaHei",sans-serif}.preview-sidebar{position:fixed;inset:0 auto 0 0;width:218px;border-right:1px solid #303033;background:#1d1d1f;padding:30px 18px}.preview-brand{display:flex;align-items:center;gap:12px;padding:0 12px;font-size:21px;font-weight:650}.brand-mark{display:grid;place-items:center;width:37px;height:37px;background:#5145eb;border-radius:12px;color:#fff}.preview-brand div>span{display:block;letter-spacing:2.8px;font-size:9px;color:#88888f;margin-top:3px}.nav-caption{margin:42px 14px 12px;font-size:11px;letter-spacing:2px;color:#77777e}.second-caption{margin-top:32px}.nav-item{display:flex;align-items:center;gap:12px;padding:13px 15px;margin:4px 0;border-radius:9px;font-size:13px;color:#aaaab2}.nav-item.active{background:#5145eb26;color:#aaa4ff}.nav-button{width:100%;text-align:left;cursor:pointer}.nav-button:hover{background:#303033}.sidebar-note{position:absolute;bottom:32px;left:32px;font-size:12px;color:#b3b3ba}.sidebar-note small{display:block;margin:7px 0 0 13px;color:#75757e}.demo-dot{display:inline-block;width:5px;height:5px;border-radius:50%;background:#9690ff;margin-right:8px}.preview-main{margin-left:218px}.preview-topbar{height:76px;border-bottom:1px solid #303033;display:flex;align-items:center;justify-content:space-between;padding:0 42px;font-size:14px}.crumb{color:#74747c;margin-left:9px}.preview-pill{border:1px solid #5145eb66;border-radius:30px;padding:5px 12px;color:#aca6ff;font-size:11px}.preview-content{max-width:1230px;margin:auto;padding:38px 42px 54px}.page-heading{display:flex;justify-content:space-between;align-items:center;gap:20px;margin-bottom:28px}.eyebrow{font-size:10px;font-weight:600;letter-spacing:2px;color:#9790f8}.page-heading h1{font-size:25px;font-weight:600;letter-spacing:-.6px;margin:10px 0}.page-heading>div>p:last-child{font-size:13px;color:#909097}.preview-grid{display:grid;grid-template-columns:minmax(0,630px) minmax(210px,1fr);gap:36px;align-items:start}.edit-card{background:#262626;border:1px solid #414141;border-radius:17px;overflow:hidden;box-shadow:0 18px 56px #0002}.edit-card-header{padding:22px 25px;display:flex;align-items:center;justify-content:space-between;border-bottom:1px solid #414141}.edit-card-header h2{font-size:19px;font-weight:600}.key-caption{font-size:10px;letter-spacing:1px;color:#82828a}.edit-card-body{padding:24px}.mode-label{margin-top:23px}.mode-tabs{display:grid;grid-template-columns:1fr 1fr;padding:4px;background:#3b3b3b;border-radius:9px;font-size:14px;text-align:center;color:#c5c5cd}.mode-tabs>span{padding:9px}.mode-tabs .selected{background:#252525;border-radius:6px;color:#b2adff}.mode-hint{font-size:12px;color:#8f8f95;margin:9px 0 22px}.edit-card-footer{display:flex;gap:10px;align-items:center;padding:17px 23px;border-top:1px solid #414141}.edit-card-footer>p{margin-right:auto;font-size:11px;color:#b2adff;max-width:260px}.preview-guide{padding-top:7px}.guide-status{display:flex;gap:7px;align-items:center;color:#b2adff;font-size:11px;margin-bottom:22px}.status-indicator{height:6px;width:6px;border-radius:50%;background:#a59dff}.status-indicator.closed{background:#e4b565}.preview-guide h2{font-size:17px;font-weight:600}.guide-steps{padding:0;list-style:none;margin:23px 0}.guide-steps li{display:flex;gap:15px;margin-bottom:25px}.guide-steps b{font-weight:500;font-size:11px;font-family:monospace;color:#777581;margin-top:3px}.guide-steps strong{font-weight:500;font-size:13px}.guide-steps p{font-size:12px;line-height:1.8;color:#93939b;margin-top:6px}.guide-note{display:flex;align-items:flex-start;gap:10px;border-top:1px solid #333338;padding-top:20px}.guide-note svg{flex-shrink:0;color:#9b95ee;margin-top:3px}.guide-note p{font-size:12px;line-height:1.9;color:#9998a4}.saved-order{margin-top:30px;padding:21px;border:1px solid #333338;border-radius:12px;background:#1b1b1e}.saved-order h3{font-size:13px;margin:10px 0 17px}.saved-order>div{display:flex;gap:12px;font-size:12px;color:#a9a9b1;margin-top:10px}.saved-order>div>span{color:#77727e;font:11px monospace}.preview-shell button:focus-visible{outline:2px solid #aca6ff;outline-offset:3px}@media(max-width:1150px){.preview-sidebar{width:175px}.preview-main{margin-left:175px}.preview-content{padding:30px 24px}.preview-grid{gap:24px;grid-template-columns:minmax(0,1fr) 215px}.page-heading h1{font-size:22px}}@media(max-width:900px){.preview-sidebar{display:none}.preview-main{margin:0}.preview-guide{display:none}.preview-grid{display:block;max-width:650px;margin:auto}.page-heading{max-width:650px;margin:0 auto 25px}.page-heading h1{font-size:20px}.preview-topbar{padding:0 24px}}@media(max-width:550px){.preview-content{padding:24px 12px}.page-heading{align-items:flex-start;flex-direction:column;gap:16px}.edit-card-body{padding:16px}.edit-card-footer{padding:16px}.edit-card-footer>p{max-width:150px}.edit-card-header{padding:18px}.preview-topbar{height:60px}}
</style>
