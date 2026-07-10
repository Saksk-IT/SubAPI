import type { OpenAIFastPolicyRule } from "@/api/admin/settings";

export function appendOpenAIFastPolicyUserID(
  rules: ReadonlyArray<OpenAIFastPolicyRule>,
  ruleIndex: number,
): OpenAIFastPolicyRule[] {
  return updateRuleUserIDs(rules, ruleIndex, (userIDs) => [...userIDs, 0]);
}

export function replaceOpenAIFastPolicyUserID(
  rules: ReadonlyArray<OpenAIFastPolicyRule>,
  ruleIndex: number,
  userIDIndex: number,
  value: number,
): OpenAIFastPolicyRule[] {
  return updateRuleUserIDs(rules, ruleIndex, (userIDs) =>
    userIDs.map((userID, index) =>
      index === userIDIndex ? value : userID,
    ),
  );
}

export function removeOpenAIFastPolicyUserID(
  rules: ReadonlyArray<OpenAIFastPolicyRule>,
  ruleIndex: number,
  userIDIndex: number,
): OpenAIFastPolicyRule[] {
  return updateRuleUserIDs(rules, ruleIndex, (userIDs) =>
    userIDs.filter((_, index) => index !== userIDIndex),
  );
}

function updateRuleUserIDs(
  rules: ReadonlyArray<OpenAIFastPolicyRule>,
  ruleIndex: number,
  update: (userIDs: number[]) => number[],
): OpenAIFastPolicyRule[] {
  return rules.map((rule, index) => {
    if (index !== ruleIndex) return rule;

    return {
      ...rule,
      user_ids: update([...(rule.user_ids ?? [])]),
    };
  });
}
