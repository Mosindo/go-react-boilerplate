import React, { type ReactNode } from "react";
import {
  KeyboardAvoidingView,
  Modal,
  Platform,
  Pressable,
  StyleSheet,
  View,
  type StyleProp,
  type ViewStyle
} from "react-native";
import { SafeAreaProvider, useSafeAreaInsets } from "react-native-safe-area-context";
import { useTheme } from "../shared/ui/theme";

type Props = {
  visible: boolean;
  onClose: () => void;
  children: ReactNode;
  /** Take (almost) the whole screen height instead of hugging the content. */
  tall?: boolean;
  testID?: string;
  accessibilityLabel: string;
  style?: StyleProp<ViewStyle>;
};

function SheetBody({
  onClose,
  children,
  tall,
  testID,
  accessibilityLabel,
  style
}: Omit<Props, "visible">) {
  const { colors, radii } = useTheme();
  const insets = useSafeAreaInsets();
  return (
    <KeyboardAvoidingView
      behavior={Platform.OS === "ios" ? "padding" : undefined}
      style={styles.root}
    >
      <Pressable
        accessibilityRole="button"
        accessibilityLabel="Close"
        onPress={onClose}
        style={[StyleSheet.absoluteFill, { backgroundColor: colors.overlay }]}
      />
      <View
        testID={testID}
        accessibilityViewIsModal
        accessibilityLabel={accessibilityLabel}
        style={[
          styles.sheet,
          {
            backgroundColor: colors.background,
            borderTopLeftRadius: radii.xl,
            borderTopRightRadius: radii.xl,
            paddingBottom: Math.max(insets.bottom, 12),
            maxHeight: tall ? "94%" : "88%"
          },
          tall ? styles.tall : null,
          style
        ]}
      >
        <View style={[styles.grabber, { backgroundColor: colors.borderStrong }]} />
        {children}
      </View>
    </KeyboardAvoidingView>
  );
}

/** Modal bottom sheet. Wraps its own SafeAreaProvider because a Modal is a separate native root. */
export function BottomSheet({ visible, onClose, ...rest }: Props) {
  return (
    <Modal
      visible={visible}
      transparent
      animationType="slide"
      onRequestClose={onClose}
      statusBarTranslucent
    >
      <SafeAreaProvider>
        <SheetBody onClose={onClose} {...rest} />
      </SafeAreaProvider>
    </Modal>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, justifyContent: "flex-end" },
  sheet: { overflow: "hidden" },
  tall: { height: "94%" },
  grabber: {
    alignSelf: "center",
    width: 40,
    height: 4,
    borderRadius: 2,
    marginTop: 8,
    marginBottom: 4
  }
});
