import { useState, useEffect, useMemo } from "react";
import type { Rule, RuleTemplate, ProxyGroupView } from "../types";
import type { RuleValidationError } from "../types";
import { getRules, createRule, updateRule, deleteRule, reorderRules, validateRules, getRuleTemplates, applyRuleTemplate } from "../api/client";
import { PanelTitle, EmptyState } from "./shared";

interface RuleEditorViewProps {
  groups: ProxyGroupView[];
  busy: boolean;
}

const RULE_TYPES = [
  "DOMAIN", "DOMAIN-SUFFIX", "DOMAIN-KEYWORD",
  "GEOIP", "IP-CIDR", "IP-CIDR6",
  "SRC-IP-CIDR", "DST-PORT",
  "PROCESS-NAME", "PROCESS-PATH",
  "MATCH", "AND", "OR", "NOT",
];

export function RuleEditorView({ groups, busy }: RuleEditorViewProps) {
  const [rules, setRules] = useState<Rule[]>([]);
  const [templates, setTemplates] = useState<RuleTemplate[]>([]);
  const [errors, setErrors] = useState<RuleValidationError[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [showTemplateModal, setShowTemplateModal] = useState(false);
  const [editingRule, setEditingRule] = useState<Partial<Rule> | null>(null);
  const [selectedCategory, setSelectedCategory] = useState<string>("all");

  const groupNames = useMemo(() =>
    [...new Set(groups.map(g => g.name))].sort(),
  [groups]);

  const loadRules = async () => {
    try {
      const [r, t] = await Promise.all([getRules(), getRuleTemplates()]);
      setRules(r);
      setTemplates(t);
    } catch { /* handled by parent */ }
    setLoading(false);
  };

  useEffect(() => { loadRules(); }, []);

  const categories = useMemo(() =>
    [...new Set(templates.map(t => t.category))],
  [templates]);
  const filteredTemplates = selectedCategory === "all"
    ? templates
    : templates.filter(t => t.category === selectedCategory);

  const openEditModal = (rule?: Rule) => {
    setEditingRule(rule || { type: "DOMAIN-SUFFIX", enabled: true, priority: 100 });
    setShowModal(true);
  };

  const saveRule = async () => {
    if (!editingRule) return;
    if (editingRule.id) {
      await updateRule(editingRule.id, editingRule);
    } else {
      await createRule(editingRule);
    }
    setShowModal(false);
    setEditingRule(null);
    await loadRules();
  };

  const removeRule = async (id: string) => {
    await deleteRule(id);
    await loadRules();
  };

  const handleReorder = async (ids: string[]) => {
    await reorderRules(ids);
    await loadRules();
  };

  const applyTemplate = async (tid: string) => {
    await applyRuleTemplate(tid);
    setShowTemplateModal(false);
    await loadRules();
  };

  if (loading) {
    return <div className="text-center text-brand-500 py-10">加载中...</div>;
  }

  return (
    <div className="flex flex-col gap-4">
      <PanelTitle title="分流规则" meta={`${rules.length} 条规则`} />

      {/* Toolbar */}
      <div className="flex gap-2">
        <button onClick={() => openEditModal()} disabled={busy}
          className="rounded-xl bg-[#dc7f69] px-4 py-2 text-sm text-white hover:bg-[#cd705d] disabled:opacity-50">
          + 新建规则
        </button>
        <button onClick={() => setShowTemplateModal(true)}
          className="rounded-xl bg-[#fff4ef] px-4 py-2 text-sm text-brand-700 hover:bg-[#ffe4dc]">
          模板库
        </button>
      </div>

      {/* Rule list */}
      {rules.length === 0
        ? <EmptyState message="还没有规则。创建规则或使用模板快速开始" />
        : <div className="rounded-3xl bg-[#fffaf7] shadow-[0_12px_32px_rgb(124_76_62/0.06)] overflow-hidden">
            {[...rules].sort((a,b) => a.priority - b.priority).map((rule, idx) => (
              <div key={rule.id} className="flex items-center gap-3 px-4 py-3 border-b border-[rgb(112_76_65/0.06)] last:border-b-0">
                <span className="text-xs text-brand-400 w-6">{idx + 1}</span>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="inline-block px-2 py-0.5 rounded-full text-xs font-bold bg-[#fff4ef] text-brand-700">
                      {rule.type}
                    </span>
                    <span className="text-sm text-brand-900 truncate">
                      {rule.type === "MATCH" ? "所有流量" : rule.value}
                    </span>
                  </div>
                  <div className="text-xs text-brand-400 mt-0.5">
                    → {rule.targetGroup || "未设置目标"}
                  </div>
                </div>
                <div className="flex gap-1">
                  <button onClick={() => openEditModal(rule)}
                    className="text-xs px-2 py-1 rounded-lg hover:bg-[#fff4ef] text-brand-700">编辑</button>
                  <button onClick={() => rule.id && removeRule(rule.id)}
                    className="text-xs px-2 py-1 rounded-lg hover:bg-[#ffe7df] text-[#8f321f]">删除</button>
                </div>
              </div>
            ))}
          </div>}

      {/* Add/Edit Modal */}
      {showModal && editingRule && (
        <div className="fixed inset-0 bg-black/20 flex items-center justify-center z-50" onClick={() => { setShowModal(false); setEditingRule(null); }}>
          <div className="bg-white rounded-3xl p-6 w-[480px] max-h-[80vh] overflow-y-auto shadow-2xl" onClick={e => e.stopPropagation()}>
            <h3 className="text-lg font-bold text-brand-900 mb-4">
              {editingRule.id ? "编辑规则" : "新建规则"}
            </h3>
            <div className="space-y-3">
              <div>
                <label className="text-xs text-brand-500 mb-1 block">规则类型</label>
                <select value={editingRule.type} onChange={e => setEditingRule({...editingRule, type: e.target.value})}
                  className="w-full rounded-xl border-0 bg-[#fff5f0] px-3 py-2 text-sm shadow-[inset_0_0_0_1px_rgb(112_76_65/0.1)]">
                  {RULE_TYPES.map(t => <option key={t} value={t}>{t}</option>)}
                </select>
              </div>
              {editingRule.type && editingRule.type !== "MATCH" && (
                <div>
                  <label className="text-xs text-brand-500 mb-1 block">匹配值</label>
                  <input value={editingRule.value || ""} onChange={e => setEditingRule({...editingRule, value: e.target.value})}
                    className="w-full rounded-xl border-0 bg-[#fff5f0] px-3 py-2 text-sm shadow-[inset_0_0_0_1px_rgb(112_76_65/0.1)]"
                    placeholder={editingRule.type.startsWith("DOMAIN") ? "example.com" : editingRule.type === "GEOIP" ? "CN" : "值"} />
                </div>
              )}
              <div>
                <label className="text-xs text-brand-500 mb-1 block">目标代理组</label>
                <select value={editingRule.targetGroup || ""} onChange={e => setEditingRule({...editingRule, targetGroup: e.target.value})}
                  className="w-full rounded-xl border-0 bg-[#fff5f0] px-3 py-2 text-sm shadow-[inset_0_0_0_1px_rgb(112_76_65/0.1)]">
                  <option value="">选择代理组...</option>
                  {groupNames.map(n => <option key={n} value={n}>{n}</option>)}
                  <option value="DIRECT">DIRECT（直连）</option>
                  <option value="REJECT">REJECT（拒绝）</option>
                  <option value="PROXY">PROXY（代理）</option>
                </select>
              </div>
              <div>
                <label className="text-xs text-brand-500 mb-1 block">描述（可选）</label>
                <input value={editingRule.description || ""} onChange={e => setEditingRule({...editingRule, description: e.target.value})}
                  className="w-full rounded-xl border-0 bg-[#fff5f0] px-3 py-2 text-sm shadow-[inset_0_0_0_1px_rgb(112_76_65/0.1)]"
                  placeholder="用户备注" />
              </div>
            </div>
            <div className="flex gap-2 mt-6 justify-end">
              <button onClick={() => { setShowModal(false); setEditingRule(null); }}
                className="rounded-xl bg-[#fff4ef] px-4 py-2 text-sm text-brand-700">取消</button>
              <button onClick={saveRule} disabled={!editingRule.type || (!editingRule.value && editingRule.type !== "MATCH") || !editingRule.targetGroup}
                className="rounded-xl bg-[#dc7f69] px-4 py-2 text-sm text-white hover:bg-[#cd705d] disabled:opacity-50">保存</button>
            </div>
          </div>
        </div>
      )}

      {/* Template Modal */}
      {showTemplateModal && (
        <div className="fixed inset-0 bg-black/20 flex items-center justify-center z-50" onClick={() => setShowTemplateModal(false)}>
          <div className="bg-white rounded-3xl p-6 w-[560px] max-h-[80vh] overflow-y-auto shadow-2xl" onClick={e => e.stopPropagation()}>
            <h3 className="text-lg font-bold text-brand-900 mb-4">规则模板库</h3>
            <div className="flex gap-2 mb-3 flex-wrap">
              <button onClick={() => setSelectedCategory("all")}
                className={`rounded-xl px-3 py-1 text-xs ${selectedCategory === "all" ? "bg-[#dc7f69] text-white" : "bg-[#fff4ef] text-brand-700"}`}>全部</button>
              {categories.map(c => (
                <button key={c} onClick={() => setSelectedCategory(c)}
                  className={`rounded-xl px-3 py-1 text-xs ${selectedCategory === c ? "bg-[#dc7f69] text-white" : "bg-[#fff4ef] text-brand-700"}`}>
                  {c === "ai" ? "🤖 AI" : c === "streaming" ? "🎬 流媒体" : c === "social" ? "💬 社交" : c === "work" ? "💼 工作" : c === "china" ? "🇨🇳 直连" : c}
                </button>
              ))}
            </div>
            <div className="space-y-2">
              {filteredTemplates.map(tmpl => (
                <div key={tmpl.id} className="bg-[#fffaf7] rounded-2xl p-4 flex justify-between items-center">
                  <div>
                    <div className="font-bold text-sm text-brand-900">{tmpl.name}</div>
                    <div className="text-xs text-brand-500">{tmpl.description}</div>
                  </div>
                  <button onClick={() => applyTemplate(tmpl.id)}
                    className="rounded-xl bg-[#dc7f69] px-3 py-1.5 text-xs text-white hover:bg-[#cd705d] shrink-0 ml-4">应用</button>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
