import React, { forwardRef, useState } from "react";
import { View, type TextInput } from "react-native";
import { Button } from "../ui/Button";
import { Input, type InputProps } from "../ui/Input";
import { type Theme } from "../ui/theme";
import { useThemedStyles } from "../ui/useThemedStyles";

export type PasswordInputProps = Omit<InputProps, "secureTextEntry" | "autoCapitalize" | "autoCorrect">;

const makeStyles = (t: Theme) => ({
  wrap: { width: "100%" as const, gap: t.spacing.xxs },
  toggle: { alignSelf: "flex-end" as const }
});

/** Password field with a show/hide control. */
export const PasswordInput = forwardRef<TextInput, PasswordInputProps>(function PasswordInput(props, ref) {
  const styles = useThemedStyles(makeStyles);
  const [visible, setVisible] = useState(false);
  return (
    <View style={styles.wrap}>
      <Input
        autoCapitalize="none"
        autoCorrect={false}
        ref={ref}
        secureTextEntry={!visible}
        textContentType={props.textContentType ?? "password"}
        {...props}
      />
      <Button
        accessibilityLabel={visible ? "Hide password" : "Show password"}
        label={visible ? "Hide" : "Show"}
        onPress={() => setVisible((value) => !value)}
        size="sm"
        style={styles.toggle}
        testID={props.testID ? `${props.testID}-toggle` : undefined}
        variant="ghost"
      />
    </View>
  );
});
