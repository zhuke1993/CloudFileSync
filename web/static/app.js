// ====================================
// CloudFileSync - 增强交互体验
// ====================================

// 全局状态
let currentConfig = {
    watch_dir: '',
    delay_time: 5,
    providers: []
};

let serviceRunning = false;
let editingProviderIndex = null;

// 初始化
document.addEventListener('DOMContentLoaded', function() {
    // 添加页面加载动画
    document.body.style.opacity = '0';
    setTimeout(() => {
        document.body.style.transition = 'opacity 0.5s ease';
        document.body.style.opacity = '1';
    }, 100);

    loadConfig();
    loadServiceStatus();
    setupEventListeners();
    setupKeyboardShortcuts();
    setupFormValidation();
});

// 设置事件监听
function setupEventListeners() {
    // 添加云盘按钮
    document.getElementById('btnAddProvider').addEventListener('click', openProviderModal);

    // 保存配置按钮
    document.getElementById('btnSave').addEventListener('click', saveConfig);

    // 重置按钮
    document.getElementById('btnReset').addEventListener('click', () => {
        loadConfig();
        showToast('配置已重置', 'info');
    });

    // 启动服务
    document.getElementById('btnStart').addEventListener('click', startService);

    // 停止服务
    document.getElementById('btnStop').addEventListener('click', stopService);

    // 清空日志
    document.getElementById('btnClearLog').addEventListener('click', clearLog);

    // 模态框关闭
    document.querySelector('.modal-close').addEventListener('click', closeProviderModal);

    // 点击模态框外部关闭
    window.addEventListener('click', function(e) {
        const modal = document.getElementById('providerModal');
        if (e.target === modal) {
            closeProviderModal();
        }
    });

    // 云盘类型选择
    document.getElementById('providerType').addEventListener('change', function() {
        const type = this.value;
        const configs = ['aliyun', 'baidu', '115', 'onedrive'];

        configs.forEach(configType => {
            const configElement = document.getElementById(configType + 'Config');
            if (configElement) {
                configElement.style.display = type === configType ? 'block' : 'none';
            }
        });

        // 添加动画效果
        const activeConfig = document.getElementById(type + 'Config');
        if (activeConfig && type) {
            activeConfig.style.animation = 'fadeIn 0.3s ease';
        }
    });

    // 提交云盘表单
    document.getElementById('providerForm').addEventListener('submit', function(e) {
        e.preventDefault();
        addProvider();
    });

    // 验证云盘
    document.getElementById('btnVerify').addEventListener('click', verifyProvider);

    // 选择目录按钮
    document.getElementById('btnBrowse').addEventListener('click', function() {
        document.getElementById('dirSelector').click();
    });

    // 目录选择器变化事件
    document.getElementById('dirSelector').addEventListener('change', function(e) {
        const files = e.target.files;
        if (files && files.length > 0) {
            // 从第一个文件的路径中提取目录路径
            const firstFile = files[0];
            const fullPath = firstFile.webkitRelativePath || firstFile.name;
            const dirPath = fullPath.substring(0, fullPath.indexOf('/'));

            // 如果有父目录信息，使用完整的目录路径
            if (firstFile.webkitRelativePath) {
                const pathParts = firstFile.webkitRelativePath.split('/');
                pathParts.pop(); // 移除文件名
                const selectedDir = pathParts.join('/');
                document.getElementById('watchDirInput').value = '/' + selectedDir;
                addLog('已选择目录: /' + selectedDir, 'success');
                showToast('目录已选择', 'success');
            }
        }
    });

    // 输入框焦点动画
    document.querySelectorAll('.form-group input, .form-group select').forEach(input => {
        input.addEventListener('focus', function() {
            this.parentElement.classList.add('focused');
        });

        input.addEventListener('blur', function() {
            this.parentElement.classList.remove('focused');
        });
    });

    // 按钮点击波纹效果
    document.querySelectorAll('.btn').forEach(button => {
        button.addEventListener('click', function(e) {
            if (this.disabled) return;

            const ripple = document.createElement('span');
            const rect = this.getBoundingClientRect();
            const size = Math.max(rect.width, rect.height);
            const x = e.clientX - rect.left - size / 2;
            const y = e.clientY - rect.top - size / 2;

            ripple.style.cssText = `
                position: absolute;
                width: ${size}px;
                height: ${size}px;
                left: ${x}px;
                top: ${y}px;
                background: rgba(255, 255, 255, 0.3);
                border-radius: 50%;
                transform: scale(0);
                animation: ripple 0.6s ease-out;
                pointer-events: none;
            `;

            this.style.position = 'relative';
            this.style.overflow = 'hidden';
            this.appendChild(ripple);

            setTimeout(() => ripple.remove(), 600);
        });
    });
}

// 设置键盘快捷键
function setupKeyboardShortcuts() {
    document.addEventListener('keydown', function(e) {
        // ESC 关闭模态框
        if (e.key === 'Escape') {
            closeProviderModal();
        }

        // Ctrl/Cmd + S 保存配置
        if ((e.ctrlKey || e.metaKey) && e.key === 's') {
            e.preventDefault();
            saveConfig();
        }
    });
}

// 设置表单验证
function setupFormValidation() {
    // 监听目录输入
    const watchDirInput = document.getElementById('watchDirInput');
    watchDirInput.addEventListener('input', function() {
        const value = this.value.trim();
        if (value && !value.startsWith('/')) {
            showInputError(this, '请输入绝对路径（以 / 开头）');
        } else {
            clearInputError(this);
        }
    });

    // 监听延迟时间输入
    const delayTimeInput = document.getElementById('delayTime');
    delayTimeInput.addEventListener('input', function() {
        const value = parseInt(this.value);
        if (value < 1 || value > 60) {
            showInputError(this, '延迟时间必须在 1-60 秒之间');
        } else {
            clearInputError(this);
        }
    });
}

// 显示输入错误
function showInputError(input, message) {
    const existingError = input.parentElement.querySelector('.input-error');
    if (existingError) return;

    const error = document.createElement('div');
    error.className = 'input-error';
    error.style.cssText = `
        color: var(--danger-color);
        font-size: 0.875em;
        margin-top: 4px;
        animation: fadeIn 0.3s ease;
    `;
    error.textContent = message;

    input.style.borderColor = 'var(--danger-color)';
    input.parentElement.appendChild(error);
}

// 清除输入错误
function clearInputError(input) {
    const existingError = input.parentElement.querySelector('.input-error');
    if (existingError) {
        existingError.remove();
    }
    input.style.borderColor = '';
}

// 加载配置（带加载状态）
async function loadConfig() {
    try {
        addLog('正在加载配置...', 'info');

        const response = await fetch('/api/config');
        const result = await response.json();

        if (result.code === 0) {
            currentConfig = result.data;
            updateUI();
            renderProviders();
            addLog('配置加载成功', 'success');
        } else {
            throw new Error(result.message);
        }
    } catch (error) {
        addLog('加载配置失败: ' + error.message, 'error');
        showToast('加载配置失败', 'error');
    }
}

// 更新界面（带动画）
function updateUI() {
    const watchDirInput = document.getElementById('watchDirInput');
    const delayTimeInput = document.getElementById('delayTime');

    // 添加淡入动画
    animateValue(watchDirInput, currentConfig.watch_dir || '');
    animateValue(delayTimeInput, currentConfig.delay_time || 5);
}

// 数值/文本变化动画
function animateValue(element, newValue) {
    element.style.transition = 'all 0.3s ease';
    element.style.opacity = '0';
    element.style.transform = 'translateX(-10px)';

    setTimeout(() => {
        element.value = newValue;
        element.style.opacity = '1';
        element.style.transform = 'translateX(0)';
    }, 150);
}

// 渲染云盘列表（带动画）
function renderProviders() {
    const container = document.getElementById('providersList');
    container.innerHTML = '';

    if (currentConfig.providers.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <div class="empty-state-icon">☁️</div>
                <div class="empty-state-text">还没有添加云盘配置</div>
                <button class="btn btn-primary" onclick="openProviderModal()" style="margin-top: 16px;">
                    立即添加
                </button>
            </div>
        `;
        return;
    }

    currentConfig.providers.forEach((provider, index) => {
        const item = createProviderItem(provider, index);
        container.appendChild(item);
    });
}

// 创建云盘列表项
function createProviderItem(provider, index) {
    const div = document.createElement('div');
    div.className = 'provider-item' + (provider.enable ? '' : ' disabled');
    div.style.animation = `fadeInUp 0.4s ease ${index * 0.1}s backwards`;

    const icons = {
        aliyun: '☁️',
        baidu: '📦',
        '115': '💎',
        onedrive: '🌐'
    };

    const typeNames = {
        aliyun: '阿里云盘',
        baidu: '百度网盘',
        '115': '115网盘',
        onedrive: 'OneDrive'
    };

    const icon = icons[provider.type] || '☁️';
    const typeName = typeNames[provider.type] || provider.type;

    div.innerHTML = `
        <div class="provider-header">
            <div class="provider-title">
                <span class="provider-icon">${icon}</span>
                <span>${provider.name || typeName}</span>
                <span class="provider-badge">${provider.enable ? '已启用' : '已禁用'}</span>
            </div>
            <div class="provider-actions">
                <button class="btn btn-secondary btn-small" onclick="toggleProvider(${index})">
                    ${provider.enable ?
                        '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="6" y="4" width="4" height="16"></rect><rect x="14" y="4" width="4" height="16"></rect></svg> 禁用' :
                        '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"></polygon></svg> 启用'}
                </button>
                <button class="btn btn-secondary btn-small" onclick="editProvider(${index})">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path></svg>
                    编辑
                </button>
                <button class="btn btn-danger btn-small" onclick="confirmDeleteProvider(${index}, '${provider.name || typeName}')">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
                    删除
                </button>
            </div>
        </div>
        <div class="provider-info">
            <div class="info-item">
                <span class="info-label">类型</span>
                <span class="info-value">${typeName}</span>
            </div>
            <div class="info-item">
                <span class="info-label">目标目录</span>
                <span class="info-value">${provider.target || '/'}</span>
            </div>
            <div class="info-item">
                <span class="info-label">Token</span>
                <span class="info-value">${maskToken(provider.tokens?.access_token || '')}</span>
            </div>
        </div>
    `;

    return div;
}

// 掩码 Token
function maskToken(token) {
    if (!token || token.length < 8) return '***';
    return token.substring(0, 4) + '****' + token.substring(token.length - 4);
}

// 保存配置（带加载状态）
async function saveConfig() {
    const saveBtn = document.getElementById('btnSave');
    const originalText = saveBtn.innerHTML;

    // 显示加载状态
    saveBtn.disabled = true;
    saveBtn.innerHTML = '<span class="loading-spinner"></span> 保存中...';

    // 获取基本配置
    const watchDir = document.getElementById('watchDirInput').value.trim();
    const delayTime = parseInt(document.getElementById('delayTime').value);

    if (!watchDir) {
        showToast('请输入监听目录', 'error');
        resetSaveButton(saveBtn, originalText);
        return;
    }

    if (delayTime < 1 || delayTime > 60) {
        showToast('延迟时间必须在 1-60 秒之间', 'error');
        resetSaveButton(saveBtn, originalText);
        return;
    }

    currentConfig.watch_dir = watchDir;
    currentConfig.delay_time = delayTime;

    try {
        const response = await fetch('/api/config/save', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(currentConfig)
        });

        const result = await response.json();

        if (result.code === 0) {
            showToast('配置保存成功', 'success');
            addLog('配置已保存', 'success');
            loadServiceStatus();
        } else {
            showToast('保存失败: ' + result.message, 'error');
        }
    } catch (error) {
        showToast('保存失败: ' + error.message, 'error');
        addLog('保存配置失败: ' + error.message, 'error');
    } finally {
        resetSaveButton(saveBtn, originalText);
    }
}

// 重置保存按钮
function resetSaveButton(button, originalText) {
    setTimeout(() => {
        button.disabled = false;
        button.innerHTML = originalText;
    }, 500);
}

// 加载服务状态
async function loadServiceStatus() {
    try {
        const response = await fetch('/api/service/status');
        const result = await response.json();

        if (result.code === 0) {
            serviceRunning = result.data.running;
            updateServiceStatusUI(result.data);
        }
    } catch (error) {
        addLog('获取服务状态失败: ' + error.message, 'error');
    }
}

// 更新服务状态界面
function updateServiceStatusUI(data) {
    const statusBadge = document.getElementById('serviceStatus');
    const watchDir = document.getElementById('watchDir');
    const btnStart = document.getElementById('btnStart');
    const btnStop = document.getElementById('btnStop');

    if (data.running) {
        statusBadge.textContent = '运行中';
        statusBadge.className = 'status-badge running';
        btnStart.disabled = true;
        btnStop.disabled = false;
    } else {
        statusBadge.textContent = '已停止';
        statusBadge.className = 'status-badge stopped';
        btnStart.disabled = false;
        btnStop.disabled = true;
    }

    watchDir.textContent = data.watchDir || '-';
}

// 启动服务
async function startService() {
    const btnStart = document.getElementById('btnStart');
    btnStart.disabled = true;
    btnStart.innerHTML = '<span class="loading-spinner"></span> 启动中...';

    try {
        const response = await fetch('/api/service/start', {
            method: 'POST'
        });

        const result = await response.json();

        if (result.code === 0) {
            showToast('服务启动成功', 'success');
            addLog('服务已启动', 'success');
            loadServiceStatus();
        } else {
            showToast('启动失败: ' + result.message, 'error');
        }
    } catch (error) {
        showToast('启动失败: ' + error.message, 'error');
        addLog('启动服务失败: ' + error.message, 'error');
    } finally {
        btnStart.innerHTML = '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"></polygon></svg> 启动服务';
    }
}

// 停止服务
async function stopService() {
    const btnStop = document.getElementById('btnStop');
    btnStop.disabled = true;
    btnStop.innerHTML = '<span class="loading-spinner"></span> 停止中...';

    try {
        const response = await fetch('/api/service/stop', {
            method: 'POST'
        });

        const result = await response.json();

        if (result.code === 0) {
            showToast('服务停止成功', 'success');
            addLog('服务已停止', 'warning');
            loadServiceStatus();
        } else {
            showToast('停止失败: ' + result.message, 'error');
        }
    } catch (error) {
        showToast('停止失败: ' + error.message, 'error');
        addLog('停止服务失败: ' + error.message, 'error');
    } finally {
        btnStop.innerHTML = '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="6" y="4" width="4" height="16"></rect><rect x="14" y="4" width="4" height="16"></rect></svg> 停止服务';
    }
}

// 打开添加云盘模态框
function openProviderModal() {
    editingProviderIndex = null;
    const modal = document.getElementById('providerModal');
    const modalTitle = modal.querySelector('h3');

    modalTitle.innerHTML = '添加云盘';

    document.getElementById('providerForm').reset();
    document.querySelectorAll('.provider-config').forEach(config => {
        config.style.display = 'none';
    });
    modal.classList.add('show');
    document.body.style.overflow = 'hidden';

    // 聚焦到第一个输入框
    setTimeout(() => {
        document.getElementById('providerType').focus();
    }, 100);
}

// 关闭模态框
function closeProviderModal() {
    const modal = document.getElementById('providerModal');
    modal.classList.remove('show');
    document.body.style.overflow = '';
}

// 验证云盘
async function verifyProvider() {
    const type = document.getElementById('providerType').value;
    const btnVerify = document.getElementById('btnVerify');
    const originalText = btnVerify.innerHTML;

    if (!type) {
        showToast('请选择云盘类型', 'error');
        return;
    }

    let tokens = {};

    if (type === 'aliyun') {
        const accessToken = document.getElementById('aliyunAccessToken').value.trim();
        const driveId = document.getElementById('aliyunDriveId').value.trim();

        if (!accessToken || !driveId) {
            showToast('请填写完整的阿里云盘配置', 'error');
            return;
        }

        tokens = {
            access_token: accessToken,
            drive_id: driveId
        };
    } else if (type === 'baidu') {
        const accessToken = document.getElementById('baiduAccessToken').value.trim();

        if (!accessToken) {
            showToast('请填写百度网盘 Access Token', 'error');
            return;
        }

        tokens = {
            access_token: accessToken
        };
    } else {
        showToast('暂不支持此云盘类型的验证', 'info');
        return;
    }

    // 显示验证中状态
    btnVerify.disabled = true;
    btnVerify.innerHTML = '<span class="loading-spinner"></span> 验证中...';

    try {
        const response = await fetch('/api/provider/verify', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                type: type,
                tokens: tokens
            })
        });

        const result = await response.json();

        if (result.code === 0) {
            showToast('验证成功', 'success');
            addLog('云盘验证成功', 'success');
        } else {
            showToast('验证失败: ' + result.message, 'error');
        }
    } catch (error) {
        showToast('验证失败: ' + error.message, 'error');
    } finally {
        btnVerify.disabled = false;
        btnVerify.innerHTML = originalText;
    }
}

// 添加云盘
function addProvider() {
    const type = document.getElementById('providerType').value;
    const name = document.getElementById('providerName').value.trim();
    const target = document.getElementById('providerTarget').value.trim();
    const enable = document.getElementById('providerEnable').checked;

    if (!type) {
        showToast('请选择云盘类型', 'error');
        return;
    }

    if (!name) {
        showToast('请输入显示名称', 'error');
        return;
    }

    let tokens = {};

    if (type === 'aliyun') {
        const accessToken = document.getElementById('aliyunAccessToken').value.trim();
        const driveId = document.getElementById('aliyunDriveId').value.trim();

        if (!accessToken || !driveId) {
            showToast('请填写完整的阿里云盘配置', 'error');
            return;
        }

        tokens = {
            access_token: accessToken,
            drive_id: driveId
        };
    } else if (type === 'baidu') {
        const accessToken = document.getElementById('baiduAccessToken').value.trim();

        if (!accessToken) {
            showToast('请填写百度网盘 Access Token', 'error');
            return;
        }

        tokens = {
            access_token: accessToken
        };
    } else if (type === '115') {
        const accessToken = document.getElementById('115AccessToken').value.trim();

        if (!accessToken) {
            showToast('请填写115网盘 Access Token', 'error');
            return;
        }

        tokens = {
            access_token: accessToken
        };
    } else if (type === 'onedrive') {
        const accessToken = document.getElementById('onedriveAccessToken').value.trim();

        if (!accessToken) {
            showToast('请填写OneDrive Access Token', 'error');
            return;
        }

        tokens = {
            access_token: accessToken,
            refresh_token: document.getElementById('onedriveRefreshToken').value.trim()
        };
    }

    const provider = {
        type: type,
        name: name,
        enable: enable,
        tokens: tokens,
        target: target || '/CloudFileSync'
    };

    if (editingProviderIndex !== null) {
        currentConfig.providers.splice(editingProviderIndex, 0, provider);
        editingProviderIndex = null;
    } else {
        currentConfig.providers.push(provider);
    }

    renderProviders();
    closeProviderModal();
    addLog('已添加云盘: ' + name, 'success');
    showToast('云盘添加成功', 'success');
}

// 切换云盘启用状态
function toggleProvider(index) {
    currentConfig.providers[index].enable = !currentConfig.providers[index].enable;
    renderProviders();
    addLog('云盘状态已更新', 'info');
    showToast(currentConfig.providers[index].enable ? '已启用' : '已禁用', 'info');
}

// 编辑云盘
function editProvider(index) {
    editingProviderIndex = index;
    const provider = currentConfig.providers[index];

    const modal = document.getElementById('providerModal');
    const modalTitle = modal.querySelector('h3');

    modalTitle.innerHTML = '编辑云盘';

    document.getElementById('providerType').value = provider.type;
    document.getElementById('providerName').value = provider.name;
    document.getElementById('providerTarget').value = provider.target;
    document.getElementById('providerEnable').checked = provider.enable;

    // 触发change事件以显示正确的配置
    document.getElementById('providerType').dispatchEvent(new Event('change'));

    if (provider.type === 'aliyun') {
        document.getElementById('aliyunAccessToken').value = provider.tokens.access_token || '';
        document.getElementById('aliyunDriveId').value = provider.tokens.drive_id || '';
    } else if (provider.type === 'baidu') {
        document.getElementById('baiduAccessToken').value = provider.tokens.access_token || '';
    } else if (provider.type === '115') {
        document.getElementById('115AccessToken').value = provider.tokens.access_token || '';
    } else if (provider.type === 'onedrive') {
        document.getElementById('onedriveAccessToken').value = provider.tokens.access_token || '';
        document.getElementById('onedriveRefreshToken').value = provider.tokens.refresh_token || '';
    }

    // 删除旧配置
    currentConfig.providers.splice(index, 1);
    renderProviders();

    modal.classList.add('show');
    document.body.style.overflow = 'hidden';
}

// 确认删除云盘
function confirmDeleteProvider(index, name) {
    showConfirmDialog(
        '删除确认',
        `确定要删除云盘「${name}」吗？此操作不可撤销。`,
        () => deleteProvider(index)
    );
}

// 删除云盘
function deleteProvider(index) {
    const name = currentConfig.providers[index].name;
    currentConfig.providers.splice(index, 1);
    renderProviders();
    addLog('已删除云盘: ' + name, 'warning');
    showToast('云盘已删除', 'success');
}

// 添加日志
function addLog(message, type = 'info') {
    const container = document.getElementById('logContainer');
    const now = new Date();
    const time = now.toLocaleTimeString('zh-CN', { hour12: false });

    const logItem = document.createElement('div');
    logItem.className = 'log-item';
    logItem.innerHTML = `
        <span class="log-time">[${time}]</span>
        <span class="log-message log-${type}">${message}</span>
    `;

    container.appendChild(logItem);
    container.scrollTop = container.scrollHeight;

    // 限制日志数量
    while (container.children.length > 100) {
        container.removeChild(container.firstChild);
    }
}

// 清空日志
function clearLog() {
    const container = document.getElementById('logContainer');
    container.style.opacity = '0';
    container.style.transform = 'translateY(10px)';

    setTimeout(() => {
        container.innerHTML = '';
        container.style.transition = 'all 0.3s ease';
        container.style.opacity = '1';
        container.style.transform = 'translateY(0)';
        addLog('日志已清空', 'info');
    }, 300);
}

// 显示 Toast
function showToast(message, type = 'info') {
    // 移除已存在的 toast
    const existingToasts = document.querySelectorAll('.toast');
    existingToasts.forEach(toast => toast.remove());

    const toast = document.createElement('div');
    toast.className = 'toast ' + type;
    toast.textContent = message;

    document.body.appendChild(toast);

    // 添加进入动画
    toast.style.animation = 'slideInRight 0.4s cubic-bezier(0.68, -0.55, 0.265, 1.55)';

    setTimeout(() => {
        toast.style.animation = 'slideInRight 0.4s ease reverse';
        setTimeout(() => toast.remove(), 400);
    }, 3000);
}

// 显示确认对话框
function showConfirmDialog(title, message, onConfirm) {
    const existingDialog = document.querySelector('.confirm-dialog');
    if (existingDialog) existingDialog.remove();

    const dialog = document.createElement('div');
    dialog.className = 'modal show confirm-dialog';
    dialog.innerHTML = `
        <div class="modal-content" style="max-width: 400px;">
            <div class="modal-header">
                <h3 style="margin: 0;">${title}</h3>
            </div>
            <p style="margin: 20px 0; color: var(--text-secondary);">${message}</p>
            <div class="modal-actions">
                <button class="btn btn-secondary" onclick="this.closest('.confirm-dialog').remove()">取消</button>
                <button class="btn btn-danger" id="confirmDeleteBtn">确定删除</button>
            </div>
        </div>
    `;

    document.body.appendChild(dialog);
    document.body.style.overflow = 'hidden';

    dialog.querySelector('#confirmDeleteBtn').addEventListener('click', () => {
        onConfirm();
        dialog.remove();
        document.body.style.overflow = '';
    });
}

// 添加加载动画样式
const style = document.createElement('style');
style.textContent = `
    .loading-spinner {
        display: inline-block;
        width: 14px;
        height: 14px;
        border: 2px solid rgba(255, 255, 255, 0.3);
        border-top-color: white;
        border-radius: 50%;
        animation: spin 0.6s linear infinite;
    }

    @keyframes spin {
        to { transform: rotate(360deg); }
    }

    @keyframes ripple {
        to {
            transform: scale(4);
            opacity: 0;
        }
    }

    .input-error {
        animation: shake 0.4s ease;
    }

    @keyframes shake {
        0%, 100% { transform: translateX(0); }
        25% { transform: translateX(-5px); }
        75% { transform: translateX(5px); }
    }
`;
document.head.appendChild(style);
