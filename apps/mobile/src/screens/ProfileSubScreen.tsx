import React, { type ReactNode } from "react";
import { View } from "react-native";
import { KeyboardScreen } from "../shared/layout/KeyboardScreen";
import { Button } from "../shared/ui/Button";
import { Text } from "../shared/ui/Text";
import { type Theme } from "../shared/ui/theme";
import { useThemedStyles } from "../shared/ui/useThemedStyles";

type Props = {
  title: string;
  subtitle?: string;
  onBack: () => void;
  children: ReactNode;
  testID?: string;
};

const makeStyles = (t: Theme) => ({
  back: { alignSelf: "flex-start" as const, marginLeft: -t.spacing.md },
  head: { gap: t.spacing.xs }
});

/** Shared frame for the pushed pages of the Profile tab. */
export function ProfileSubScreen({ children, onBack, subtitle, testID, title }: Props) {
  const styles = useThemedStyles(makeStyles);
  return (
    <KeyboardScreen edges={["top", "left", "right"]} testID={testID}>
      <Button label="Back" onPress={onBack} size="sm" style={styles.back} testID="profile-back" variant="ghost" />
      <View style={styles.head}>
        <Text accessibilityRole="header" variant="title" weight="bold">
          {title}
        </Text>
        {subtitle ? <Text tone="muted">{subtitle}</Text> : null}
      </View>
      {children}
    </KeyboardScreen>
  );
}
