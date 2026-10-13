import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useNavigation, useRoute, type RouteProp } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { ProfileDetailView } from "../features/profile/ProfileDetailView";
import { SafetyMenu } from "../features/safety/SafetyMenu";
import { usePublicProfile } from "../hooks/queries";
import type { RootStackParamList } from "../navigation/types";
import { ErrorView, LoadingView } from "../shared/feedback";
import { SafeAreaLayout } from "../shared/layout";
import { spacing } from "../shared/ui";
import { useTheme } from "../theme";

type Nav = NativeStackNavigationProp<RootStackParamList>;

export default function ProfileDetailScreen() {
  const { colors } = useTheme();
  const navigation = useNavigation<Nav>();
  const { params } = useRoute<RouteProp<RootStackParamList, "ProfileDetail">>();
  const query = usePublicProfile(params.userId);
  const [safetyOpen, setSafetyOpen] = useState(false);

  const roundButton = (icon: keyof typeof Ionicons.glyphMap, label: string, onPress: () => void, side: "left" | "right") => (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole="button"
      onPress={onPress}
      style={[styles.round, side === "left" ? styles.left : styles.right, { backgroundColor: colors.surface }]}
    >
      <Ionicons color={colors.text} name={icon} size={22} />
    </Pressable>
  );

  return (
    <SafeAreaLayout edges={["bottom"]}>
      {query.isLoading ? (
        <LoadingView />
      ) : query.isError || !query.data ? (
        <ErrorView
          actionLabel="Retour"
          message="Ce profil n'est plus disponible."
          onAction={() => navigation.goBack()}
          title="Profil introuvable"
        />
      ) : (
        <View style={styles.flex}>
          <ProfileDetailView profile={query.data} />
          {roundButton("chevron-back", "Retour", () => navigation.goBack(), "left")}
          {roundButton("ellipsis-horizontal", "Plus d'options", () => setSafetyOpen(true), "right")}
          <SafetyMenu
            name={query.data.firstName}
            onBlocked={() => navigation.popToTop()}
            onClose={() => setSafetyOpen(false)}
            userId={params.userId}
            visible={safetyOpen}
          />
        </View>
      )}
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  round: { position: "absolute", top: spacing.lg, width: 44, height: 44, borderRadius: 22, alignItems: "center", justifyContent: "center" },
  left: { left: spacing.lg },
  right: { right: spacing.lg }
});
