import React from "react";
import { View } from "react-native";
import { spacing } from "../../theme";
import { Card, Icon, Screen, Text } from "../../ui";

const TIPS = [
  ["Take your time", "There is no rush. Chat until you feel comfortable before sharing personal details."],
  ["Protect your details", "Keep your address, workplace, financial information and passwords private."],
  [
    "Meet somewhere public",
    "Choose a busy place for a first meeting, tell a friend where you are going, and arrange your own way home."
  ],
  [
    "Trust your instincts",
    "If something feels off, it is fine to stop talking. You can unmatch, block or report at any time."
  ],
  ["Never send money", "Anyone asking for money, gift cards or crypto is a red flag. Report them."],
  ["Consent matters", "Respect boundaries and expect the same in return. Lumen is for adults aged 18 and over."]
] as const;

export function SafetyTipsScreen() {
  return (
    <Screen scroll>
      <Text tone="muted">Good connections start with feeling safe. A few warm reminders:</Text>
      {TIPS.map(([title, body]) => (
        <Card key={title}>
          <View style={{ flexDirection: "row", gap: spacing.sm, padding: spacing.md }}>
            <Icon name="heart-circle-outline" size={26} />
            <View style={{ flex: 1, gap: 2 }}>
              <Text weight="semibold">{title}</Text>
              <Text tone="muted">{body}</Text>
            </View>
          </View>
        </Card>
      ))}
    </Screen>
  );
}
