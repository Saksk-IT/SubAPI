import { mount } from "@vue/test-utils";
import { describe, expect, it, vi } from "vitest";
import { createPinia } from "pinia";
import SubscriptionPlanCard from "../SubscriptionPlanCard.vue";

vi.mock("vue-i18n", async () => {
  const actual = await vi.importActual<typeof import("vue-i18n")>("vue-i18n");
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) =>
        ({
          "payment.days": "days",
          "payment.months": "months",
          "payment.admin.weeks": "weeks",
          "payment.planCard.quota": "Quota",
          "payment.planCard.rate": "Rate",
          "payment.planCard.dailyLimit": "Daily limit",
          "payment.planCard.weeklyLimit": "Weekly limit",
          "payment.planCard.monthlyLimit": "Monthly limit",
          "payment.planCard.peakRate": "Peak rate",
          "payment.planCard.unlimited": "Unlimited",
          "payment.planCard.models": "Models",
          "payment.subscribeNow": "Subscribe now",
          "payment.renewNow": "Renew now",
        })[key] ?? key,
    }),
  };
});

const mountPlanCard = (groupPlatform: string, validityUnit = "day") =>
  mount(SubscriptionPlanCard, {
    props: {
      plan: {
        id: 1,
        group_id: 10,
        group_platform: groupPlatform,
        name: "Pro",
        price: 10,
        amount: 1000,
        features: [],
        rate_multiplier: 1,
        validity_days: 30,
        validity_unit: validityUnit,
        supported_model_scopes: ["claude", "gemini_text", "gemini_image"],
        is_active: true,
      },
    },
    global: { plugins: [createPinia()] },
  });

describe("SubscriptionPlanCard", () => {
  it("does not show Antigravity model scopes for OpenAI plans", () => {
    const text = mountPlanCard("openai").text();

    expect(text).not.toContain("Claude");
    expect(text).not.toContain("Gemini");
    expect(text).not.toContain("Imagen");
  });

  it("shows model scopes for Antigravity plans", () => {
    const text = mountPlanCard("antigravity").text();

    expect(text).toContain("Claude");
    expect(text).toContain("Gemini");
    expect(text).toContain("Imagen");
  });

  it("formats plural validity units", () => {
    expect(mountPlanCard("openai", "weeks").text()).toContain("30weeks");
    expect(mountPlanCard("openai", "months").text()).toContain("30months");
  });
});
