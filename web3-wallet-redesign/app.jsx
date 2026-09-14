const { useEffect, useState } = React;

const Icon = ({ name, size = 17 }) => {
  const paths = {
    eye: <><path d="M2 12s3.5-6 10-6 10 6 10 6-3.5 6-10 6S2 12 2 12Z"></path><circle cx="12" cy="12" r="2.5"></circle></>,
    eyeOff: <><path d="m3 3 18 18"></path><path d="M10.6 6.2A9.7 9.7 0 0 1 12 6c7 0 10 6 10 6a17 17 0 0 1-2 2.7M6.4 6.4C3.5 8.1 2 12 2 12s3.5 6 10 6c1.2 0 2.3-.2 3.3-.6"></path></>,
    send: <><path d="m12 19 0-14"></path><path d="m7 10 5-5 5 5"></path></>,
    receive: <><path d="m12 5 0 14"></path><path d="m17 14-5 5-5-5"></path></>,
    swap: <><path d="M7 7h11l-3-3"></path><path d="M17 17H6l3 3"></path></>,
    bridge: <><path d="M4 17c2-7 14-7 16 0"></path><path d="M7 17V9m10 8V9"></path></>,
    wallet: <><path d="M3 6.5h16v12H3z"></path><path d="M3 8V5h13"></path><path d="M15 12h4"></path></>,
    market: <><path d="M4 18V9m5 9V5m5 13v-6m5 6V3"></path></>,
    trade: <><path d="M5 7h14m-4-4 4 4-4 4"></path><path d="M19 17H5m4 4-4-4 4-4"></path></>,
    pay: <><rect x="3" y="5" width="18" height="14" rx="2"></rect><path d="M3 10h18"></path></>,
    agent: <><circle cx="12" cy="12" r="8"></circle><path d="M9 10h.01M15 10h.01M9 15c2 1.3 4 1.3 6 0"></path></>,
    shield: <><path d="M12 3 5 6v5c0 4.5 2.8 8 7 10 4.2-2 7-5.5 7-10V6Z"></path><path d="m9 12 2 2 4-5"></path></>,
    lock: <><rect x="5" y="10" width="14" height="10" rx="2"></rect><path d="M8 10V7a4 4 0 0 1 8 0v3"></path></>,
    key: <><circle cx="8" cy="15" r="4"></circle><path d="m11 12 8-8m-3 3 2 2"></path></>,
    trash: <><path d="M4 7h16m-10 4v5m4-5v5M9 7l1-3h4l1 3m3 0-1 14H7L6 7"></path></>,
    settings: <><circle cx="12" cy="12" r="3"></circle><path d="M19 12a7 7 0 0 0-.1-1l2-1.5-2-3.4-2.4 1A8 8 0 0 0 15 6l-.3-2.6h-4L10.4 6a8 8 0 0 0-1.5.9l-2.4-1-2 3.4 2 1.5a7 7 0 0 0 0 2.2l-2 1.5 2 3.4 2.4-1c.5.4 1 .7 1.5.9l.3 2.6h4l.3-2.6c.6-.2 1.1-.5 1.5-.9l2.4 1 2-3.4-2-1.5c.1-.3.1-.7.1-1Z"></path></>,
    close: <><path d="m6 6 12 12M18 6 6 18"></path></>
  };
  return <svg className="icon" width={size} height={size} viewBox="0 0 24 24" aria-hidden="true">{paths[name]}</svg>;
};

const chains = [
  { id: "eth", mark: "Ξ", name: "Ethereum", sub: "Mainnet" },
  { id: "base", mark: "B", name: "Base", sub: "L2" },
  { id: "sol", mark: "S", name: "Solana", sub: "Mainnet" }
];

const assets = [
  { id: "eth", mark: "Ξ", name: "Ethereum", ticker: "ETH", amount: "2.184", value: "$7,481.20" },
  { id: "usdc", mark: "$", name: "USD Coin", ticker: "USDC", amount: "4,820.00", value: "$4,820.00" },
  { id: "sol", mark: "S", name: "Solana", ticker: "SOL", amount: "21.42", value: "$3,901.18" }
];

function Modal({ title, kicker, children, onClose }) {
  return <div className="scrim" onMouseDown={(event) => event.target === event.currentTarget && onClose()}>
    <section className="modal" role="dialog" aria-modal="true">
      <div className="modal-head">
        <div><div className="modal-kicker">{kicker}</div><h2>{title}</h2></div>
        <button className="close-button" type="button" onClick={onClose} aria-label="关闭"><Icon name="close" /></button>
      </div>
      {children}
    </section>
  </div>;
}

function SecurityModal({ locked, onLock, onUnlock, onChange, onReset, onClose }) {
  return <Modal title="钱包安全中心" kicker="Local vault · AES-256-GCM" onClose={onClose}>
    <p>密钥材料仅保存在当前设备。敏感操作需要重新验证钱包密码。</p>
    <div className="security-grid">
      <button className="security-action" type="button" onClick={locked ? onUnlock : onLock}>
        <span className="security-icon"><Icon name="lock" /></span>
        <span className="security-copy"><strong>{locked ? "解锁钱包" : "立即锁定"}</strong><span>{locked ? "恢复当前设备的钱包会话" : "清除当前页面中的密钥材料"}</span></span>
      </button>
      <button className="security-action" type="button" onClick={onChange}>
        <span className="security-icon"><Icon name="key" /></span>
        <span className="security-copy"><strong>更换密码</strong><span>重新加密本地钱包保险库</span></span>
      </button>
      <button className="security-action danger" type="button" onClick={onReset}>
        <span className="security-icon"><Icon name="trash" /></span>
        <span className="security-copy"><strong>重置钱包</strong><span>永久移除本设备的加密保险库</span></span>
      </button>
    </div>
  </Modal>;
}

function PasswordModal({ mode, onClose, onSuccess }) {
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState("");
  const changing = mode === "change";
  const submit = (event) => {
    event.preventDefault();
    if (current !== "Web3demo!") return setError("当前密码不正确。演示密码：Web3demo!");
    if (changing && !/^[A-Za-z0-9!@#$%^&*.]{8,16}$/.test(next)) return setError("新密码需要 8–16 位，仅支持常见字母、数字和符号。");
    if (changing && next !== confirm) return setError("两次输入的新密码不一致。");
    onSuccess(changing ? "密码已安全更换" : "钱包已解锁");
  };
  return <Modal title={changing ? "更换钱包密码" : "解锁钱包"} kicker="Zero-knowledge local check" onClose={onClose}>
    <p>{changing ? "验证当前密码后，使用新的随机盐与 IV 重新加密保险库。" : "密码只用于在本设备派生解密密钥，不会发送到服务器。"}</p>
    <form onSubmit={submit}>
      <label className="field"><span className="field-meta">{changing ? "当前密码" : "钱包密码"}<span>8–16 位</span></span><input autoFocus type="password" value={current} onChange={(e) => {setCurrent(e.target.value);setError("");}} placeholder="输入当前钱包密码" /></label>
      {changing && <>
        <label className="field"><span className="field-meta">新密码<span>强密码</span></span><input type="password" value={next} onChange={(e) => {setNext(e.target.value);setError("");}} placeholder="输入新钱包密码" /></label>
        <label className="field"><span className="field-meta">确认新密码</span><input type="password" value={confirm} onChange={(e) => {setConfirm(e.target.value);setError("");}} placeholder="再次输入新钱包密码" /></label>
      </>}
      <div className="form-error">{error}</div>
      <div className="form-actions"><button className="ghost-button" type="button" onClick={onClose}>取消</button><button className="pill-button" type="submit">{changing ? "确认更换" : "安全解锁"}</button></div>
    </form>
  </Modal>;
}

function ResetModal({ onClose, onConfirm }) {
  const [phrase, setPhrase] = useState("");
  return <Modal title="重置本地钱包" kicker="Destructive action" onClose={onClose}>
    <p>此操作不会影响链上资产，但会永久移除本设备的加密保险库。</p>
    <div className="warning-box">继续前请确认已经离线备份助记词。若没有备份，钱包及资产可能无法恢复。</div>
    <label className="field"><span className="field-meta">输入 RESET 确认</span><input value={phrase} onChange={(e) => setPhrase(e.target.value)} placeholder="RESET" /></label>
    <div className="form-actions"><button className="ghost-button" type="button" onClick={onClose}>返回</button><button className="pill-button danger" disabled={phrase !== "RESET"} type="button" onClick={onConfirm}>重置钱包</button></div>
  </Modal>;
}

function WalletHome({ balanceVisible, setBalanceVisible, chain, setChain, onSecurity, toast }) {
  const [chainsOpen, setChainsOpen] = useState(false);
  return <div data-screen-label="Wallet Home">
    <div className="topbar">
      <div className="brand-block"><span className="eyebrow">web3 service agent</span><span className="brand">ONCHAIN<b>/</b>OS</span></div>
      <button className="network-button" type="button" onClick={() => setChainsOpen(!chainsOpen)}>
        <span className={`chain-mark ${chain.id}`}>{chain.mark}</span><span className="network-copy"><strong>{chain.name}</strong><span>● SYNCED</span></span>
      </button>
    </div>
    {chainsOpen && <div className="chain-menu">{chains.map((item) => <button className="chain-option" type="button" key={item.id} onClick={() => {setChain(item);setChainsOpen(false);toast(`已切换到 ${item.name}`);}}><span className={`chain-mark ${item.id}`}>{item.mark}</span><span className="chain-copy">{item.name}<small>{item.sub}</small></span>{chain.id === item.id && <span className="active-label">ACTIVE</span>}</button>)}</div>}
    <section className="balance-zone">
      <div className="wallet-row"><div className="wallet-id">Vault 01 · 0x71F8...9A20</div><button className="icon-button" type="button" onClick={() => setBalanceVisible(!balanceVisible)} aria-label="切换余额显示"><Icon name={balanceVisible ? "eye" : "eyeOff"} /></button></div>
      <h1 className="balance">{balanceVisible ? "$16,202" : "$••••••"}<small>.38</small></h1>
      <div className="balance-meta"><span>Portfolio value</span><span className="gain">+$428.20 · 2.71%</span><span>24H</span></div>
    </section>
    <div className="pulse-line"></div>
    <div className="quick-grid">
      {[['send','发送'],['receive','接收'],['swap','兑换'],['bridge','跨链']].map(([icon,label],i)=><button className={`quick-action ${i===2?'primary':''}`} key={label} type="button" onClick={()=>toast(`${label}流程已打开（交互演示）`)}><Icon name={icon}/><span>{label}</span></button>)}
    </div>
    <section className="section">
      <div className="section-head"><h2>链上资产</h2><button type="button" onClick={()=>toast("资产数据已刷新")}>刷新  ↻</button></div>
      <div className="asset-list">{assets.map(asset=><div className="asset-row" key={asset.id} onClick={()=>toast(`${asset.name} 资产详情`)}><span className={`coin ${asset.id}`}>{asset.mark}</span><span className="asset-name"><strong>{asset.name}</strong><span>{asset.ticker} · {chain.name}</span></span><span className="asset-value"><strong>{balanceVisible?asset.amount:'••••'}</strong><span>{balanceVisible?asset.value:'$••••'}</span></span></div>)}</div>
    </section>
    <section className="agent-card">
      <div className="agent-head"><span className="agent-orb"><Icon name="agent"/></span><strong>Agent 风险扫描</strong><span>LIVE ANALYSIS</span></div>
      <p>当前资产分布偏向 ETH，跨链桥接暴露较低。建议保留至少 $120 等值 ETH 作为未来 7 天 Gas 储备。</p>
      <button className="agent-link" type="button" onClick={()=>toast("已生成完整资产分析")}>查看完整分析 →</button>
    </section>
    <button className="icon-button" style={{position:'absolute',top:124,right:18}} type="button" onClick={onSecurity} aria-label="打开安全中心"><Icon name="settings"/></button>
  </div>;
}

function App() {
  const [tab, setTab] = useState("wallet");
  const [balanceVisible, setBalanceVisible] = useState(true);
  const [chain, setChain] = useState(chains[0]);
  const [locked, setLocked] = useState(false);
  const [modal, setModal] = useState(null);
  const [toastText, setToastText] = useState("");
  const toast = (text) => setToastText(text);
  useEffect(() => { if (!toastText) return; const timer=setTimeout(()=>setToastText(""),2100); return ()=>clearTimeout(timer); }, [toastText]);
  const nav = [
    {id:'agent',label:'Agent',icon:'agent'}, {id:'market',label:'行情',icon:'market'}, {id:'trade',label:'交易',icon:'trade'}, {id:'pay',label:'支付',icon:'pay'}, {id:'wallet',label:'钱包',icon:'wallet'}
  ];
  const emptyCopy = {agent:['AI Agent','探索资产、交易与链上风险'],market:['实时行情','追踪主流资产和市场异动'],trade:['智能交易','在确认前预览路径、价格与 Gas'],pay:['链上支付','扫码或输入地址发起安全转账']};
  return <main className="prototype-shell" lang="zh">
    <section className="phone" aria-label="Web3 Agent 钱包交互原型">
      <div className="screen">
        <div className="statusbar"><span>09:41</span><span className="signal"><i></i> MAINNET · 18ms</span></div>
        {tab === 'wallet' ? <WalletHome balanceVisible={balanceVisible} setBalanceVisible={setBalanceVisible} chain={chain} setChain={setChain} onSecurity={()=>setModal('security')} toast={toast}/> : <div className="empty-view" data-screen-label={emptyCopy[tab][0]}><span className="empty-orbit"><Icon name={nav.find(n=>n.id===tab).icon} size={28}/></span><h2>{emptyCopy[tab][0]}</h2><p>{emptyCopy[tab][1]}。这是本轮视觉方向的导航状态演示。</p><button className="pill-button" style={{padding:'0 20px'}} type="button" onClick={()=>toast(`${emptyCopy[tab][0]}功能已响应`)}>开始探索</button></div>}
      </div>
      <nav className="nav" aria-label="主导航">{nav.map(item=><button className={tab===item.id?'active':''} type="button" key={item.id} onClick={()=>setTab(item.id)}><Icon name={item.icon}/><span>{item.label}</span></button>)}</nav>
      {modal === 'security' && <SecurityModal locked={locked} onClose={()=>setModal(null)} onLock={()=>{setLocked(true);setModal(null);toast('钱包已锁定');}} onUnlock={()=>setModal('unlock')} onChange={()=>setModal('change')} onReset={()=>setModal('reset')}/>} 
      {(modal === 'unlock' || modal === 'change') && <PasswordModal mode={modal} onClose={()=>setModal(null)} onSuccess={(text)=>{setLocked(false);setModal(null);toast(text);}}/>}
      {modal === 'reset' && <ResetModal onClose={()=>setModal(null)} onConfirm={()=>{setLocked(true);setModal(null);toast('本地钱包已重置（演示）');}}/>}
      {toastText && <div className="toast" role="status">{toastText}</div>}
    </section>
  </main>;
}

ReactDOM.createRoot(document.getElementById("root")).render(<App />);
