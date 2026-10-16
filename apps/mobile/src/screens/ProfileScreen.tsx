import React, { useState } from "react";
import { Image, Pressable, ScrollView, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { absoluteUrl } from "../api/client";
import { DeleteAccountSheet } from "../components/DeleteAccountSheet";
import { useMyProfile } from "../hooks/queries";
import { useAuth } from "../hooks/useAuth";
import type { RootStackParamList } from "../navigation/types";
import { ErrorView, LoadingView } from "../shared/feedback";
import { Header, ScreenContainer } from "../shared/layout";
import { Button, Card, Text, colors, radii, spacing } from "../shared/ui";

type Route = Exclude<keyof RootStackParamList, "Tabs" | "Thread" | "ProfileDetail">;

const MENU: readonly { route: Route; icon: React.ComponentProps<typeof Ionicons>["name"]; label: string; hint: string }[] = [
  { route: "EditProfile", icon: "create-outline", label: "Modifier mon profil", hint: "Prénom, bio, centres d'intérêt, localisation" },
  { route: "Photos", icon: "images-outline", label: "Mes photos", hint: "Ajouter, réordonner, supprimer" },
  { route: "Preferences", icon: "options-outline", label: "Préférences de rencontre", hint: "Qui, âge, distance" },
  { route: "Privacy", icon: "lock-closed-outline", label: "Confidentialité", hint: "Visibilité et distance" },
  { route: "Blocked", icon: "ban-outline", label: "Personnes bloquées", hint: "Gérer votre liste de blocage" }
];

export default function ProfileScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const { user, signOut } = useAuth();
  const me = useMyProfile();
  const [deleting, setDeleting] = useState(false);

  const photo = me.data?.photos[0];

  return (
    <ScreenContainer testID="screen-profile">
      <Header title="Mon profil" />
      {me.isLoading ? <LoadingView label="Chargement…" /> : null}
      {me.isError && !me.data ? <ErrorView message="Impossible de charger votre profil." onAction={() => void me.refetch()} /> : null}
      {me.data?.profile ? (
        <ScrollView contentContainerStyle={styles.scroll} showsVerticalScrollIndicator={false}>
          <Card padding="md" style={styles.summary}>
            {photo ? <Image source={{ uri: absoluteUrl(photo.url) }} style={styles.photo} /> : <View style={styles.photo} />}
            <View style={styles.summaryCopy}>
              <Text variant="heading" weight="bold">
                {me.data.profile.firstName}, {me.data.profile.age}
              </Text>
              <Text numberOfLines={1} tone="muted" variant="caption">
                {user?.email}
              </Text>
              {!me.data.profile.discoverable ? (
                <Text tone="primary" variant="caption" weight="semibold">
                  Profil masqué de la découverte
                </Text>
              ) : null}
            </View>
          </Card>

          <View style={styles.menu}>
            {MENU.map((item) => (
              <Pressable
                accessibilityRole="button"
                key={item.route}
                onPress={() => navigation.navigate(item.route)}
                style={({ pressed }) => [styles.menuRow, pressed ? styles.pressed : null]}
                testID={`menu-${item.route}`}
              >
                <Ionicons color={colors.primary} name={item.icon} size={22} />
                <View style={styles.menuCopy}>
                  <Text weight="semibold">{item.label}</Text>
                  <Text tone="muted" variant="caption">
                    {item.hint}
                  </Text>
                </View>
                <Ionicons color={colors.textSubtle} name="chevron-forward" size={18} />
              </Pressable>
            ))}
          </View>

          <Card padding="md" variant="accent">
            <Text weight="bold">100 % gratuit, pour toujours</Text>
            <Text tone="secondary" variant="label">
              Pas d'abonnement, pas de boost payant, pas de limite de likes, pas de publicité.
            </Text>
          </Card>

          <Button label="Se déconnecter" onPress={() => void signOut()} testID="sign-out" variant="outline" />
          <Button
            label="Supprimer mon compte"
            onPress={() => setDeleting(true)}
            testID="delete-account"
            variant="ghost"
            textStyle={styles.danger}
          />
        </ScrollView>
      ) : null}
      <DeleteAccountSheet
        onClose={() => setDeleting(false)}
        onDeleted={() => {
          setDeleting(false);
          void signOut("Votre compte a été supprimé.");
        }}
        visible={deleting}
      />
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  scroll: { gap: spacing.lg, paddingTop: spacing.lg, paddingBottom: spacing.xxxl },
  summary: { flexDirection: "row", alignItems: "center", gap: spacing.lg },
  photo: { width: 72, height: 90, borderRadius: radii.md, backgroundColor: colors.surfaceSubtle },
  summaryCopy: { flex: 1, gap: spacing.xxs },
  menu: {
    backgroundColor: colors.backgroundElevated,
    borderRadius: radii.lg,
    borderWidth: 1,
    borderColor: colors.border,
    overflow: "hidden"
  },
  menuRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    padding: spacing.lg,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: colors.border
  },
  menuCopy: { flex: 1 },
  pressed: { backgroundColor: colors.surfaceMuted },
  danger: { color: colors.danger }
});
