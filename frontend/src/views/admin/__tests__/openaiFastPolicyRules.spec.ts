import { describe, expect, it } from "vitest";

import type { OpenAIFastPolicyRule } from "@/api/admin/settings";
import {
  appendOpenAIFastPolicyUserID,
  removeOpenAIFastPolicyUserID,
  replaceOpenAIFastPolicyUserID,
} from "../openaiFastPolicyRules";

function createRule(userIds?: number[]): OpenAIFastPolicyRule {
  return {
    service_tier: "priority",
    action: "filter",
    scope: "all",
    user_ids: userIds,
  };
}

describe("OpenAI Fast/Flex policy user ID updates", () => {
  it("appends a placeholder without mutating the rules or existing user IDs", () => {
    const firstRule = createRule([7]);
    const secondRule = createRule();
    const rules = [firstRule, secondRule];

    const updated = appendOpenAIFastPolicyUserID(rules, 0);

    expect(updated).not.toBe(rules);
    expect(updated[0]).not.toBe(firstRule);
    expect(updated[0].user_ids).toEqual([7, 0]);
    expect(updated[1]).toBe(secondRule);
    expect(firstRule.user_ids).toEqual([7]);
  });

  it("replaces one user ID using copied rule and array values", () => {
    const firstRule = createRule([7, 9]);
    const rules = [firstRule];

    const updated = replaceOpenAIFastPolicyUserID(rules, 0, 1, 11);

    expect(updated).not.toBe(rules);
    expect(updated[0]).not.toBe(firstRule);
    expect(updated[0].user_ids).not.toBe(firstRule.user_ids);
    expect(updated[0].user_ids).toEqual([7, 11]);
    expect(firstRule.user_ids).toEqual([7, 9]);
  });

  it("removes one user ID without changing the source rule", () => {
    const firstRule = createRule([7, 9]);
    const rules = [firstRule];

    const updated = removeOpenAIFastPolicyUserID(rules, 0, 0);

    expect(updated[0].user_ids).toEqual([9]);
    expect(firstRule.user_ids).toEqual([7, 9]);
  });
});
