import React from "react";
import { Platform, Switch } from "react-native";
import { useTheme } from "../theme/ThemeProvider";

type Props = {
  value: boolean;
  onValueChange: (v: boolean) => void;
  accessibilityLabel: string;
  testID?: string;
};

/** Themed switch (react-native-web needs its own active thumb color prop). */
export function Toggle({ value, onValueChange, accessibilityLabel, testID }: Props) {
  const { colors } = useTheme();
  const webProps = Platform.OS === "web" ? ({ activeThumbColor: "#FFFFFF" } as object) : {};
  return (
    <Switch
      testID={testID}
      accessibilityLabel={accessibilityLabel}
      value={value}
      onValueChange={onValueChange}
      thumbColor="#FFFFFF"
      trackColor={{ true: colors.primary, false: colors.borderStrong }}
      {...webProps}
    />
  );
}
