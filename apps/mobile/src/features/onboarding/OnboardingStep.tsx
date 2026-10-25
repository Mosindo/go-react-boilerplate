import React from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import { radius, spacing, useTheme } from "../../theme";
import { IconButton, Screen, Text } from "../../shared/ui";

export const ONBOARDING_STEPS = 5;

type Props = {
  step: number;
  title: string;
  subtitle?: string;
  footer: React.ReactNode;
  children: React.ReactNode;
  testID: string;
};

export function OnboardingStep({ step, title, subtitle, footer, children, testID }: Props) {
  const { colors } = useTheme();
  const navigation = useNavigation();

  return (
    <Screen edges={["top", "bottom"]} footer={footer} padded={false} scroll testID={testID}>
      <View style={styles.header}>
        {step > 1 ? (
          <IconButton icon="chevron-back" label="Retour" onPress={() => navigation.goBack()} />
        ) : (
          <View style={styles.backSpacer} />
        )}
        <View
          accessibilityLabel={`Étape ${step} sur ${ONBOARDING_STEPS}`}
          accessibilityRole="progressbar"
          style={[styles.track, { backgroundColor: colors.border }]}
        >
          <View
            style={[styles.bar, { backgroundColor: colors.primary, width: `${(step / ONBOARDING_STEPS) * 100}%` }]}
          />
        </View>
      </View>
      <View style={styles.body}>
        <Text variant="title">{title}</Text>
        {subtitle ? <Text color="textMuted">{subtitle}</Text> : null}
        {children}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: { flexDirection: "row", alignItems: "center", gap: spacing.md, paddingHorizontal: spacing.md },
  backSpacer: { width: 44, height: 44 },
  track: { flex: 1, height: 6, borderRadius: radius.pill, overflow: "hidden" },
  bar: { height: 6, borderRadius: radius.pill },
  body: { paddingHorizontal: spacing.xl, gap: spacing.lg }
});
