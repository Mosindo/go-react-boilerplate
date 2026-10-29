import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import React from "react";
import { ScrollView, StyleSheet, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { Avatar } from "../../components/Avatar";
import { ListRow } from "../../components/ListRow";
import { ErrorView, LoadingView } from "../../components/StateViews";
import { useProfile } from "../../hooks/queries";
import { errorMessage } from "../../hooks/useAsyncAction";
import type { MainTabParamList, RootStackParamList } from "../../navigation/types";
import { useMe } from "../../state/AuthProvider";
import { fontSizes, spacing } from "../../theme/palette";
import { useTheme } from "../../theme/ThemeProvider";

type Props = CompositeScreenProps<
  BottomTabScreenProps<MainTabParamList, "Me">,
  NativeStackScreenProps<RootStackParamList>
>;

export default function ProfileScreen({ navigation }: Props) {
  const theme = useTheme();
  const me = useMe();
  const query = useProfile();
  const profile = query.data;

  return (
    <SafeAreaView edges={["top"]} style={[styles.flex, { backgroundColor: theme.background }]}>
      {query.isLoading ? (
        <LoadingView />
      ) : query.isError || !profile ? (
        <ErrorView message={errorMessage(query.error)} onRetry={() => void query.refetch()} />
      ) : (
        <ScrollView contentContainerStyle={styles.content}>
          <View style={styles.hero}>
            <Avatar name={profile.firstName} size={112} url={profile.photos[0]?.url} />
            <Text style={[styles.name, { color: theme.text }]}>
              {profile.firstName}, {profile.age}
            </Text>
            <Text style={[styles.email, { color: theme.textMuted }]}>{me.email}</Text>
            {!profile.isVisible ? (
              <Text
                style={[styles.badge, { color: theme.accent, backgroundColor: theme.surfaceMuted }]}
              >
                Profil masqué dans la découverte
              </Text>
            ) : null}
          </View>

          <View
            style={[styles.group, { backgroundColor: theme.surface, borderColor: theme.border }]}
          >
            <ListRow
              icon="eye-outline"
              onPress={() => navigation.navigate("ProfileDetail", { userId: me.id })}
              title="Voir mon profil"
            />
            <ListRow
              icon="create-outline"
              onPress={() => navigation.navigate("EditProfile")}
              title="Modifier mes informations"
            />
            <ListRow
              icon="images-outline"
              onPress={() => navigation.navigate("Photos")}
              subtitle={`${profile.photos.length}/6`}
              title="Mes photos"
            />
            <ListRow
              icon="options-outline"
              onPress={() => navigation.navigate("Preferences")}
              title="Préférences de rencontre"
            />
            <ListRow
              icon="shield-checkmark-outline"
              onPress={() => navigation.navigate("Settings")}
              title="Confidentialité et compte"
            />
          </View>

          <Text style={[styles.free, { color: theme.textMuted }]}>
            Sillage est 100 % gratuit : aucun abonnement, aucune limite de likes, aucune publicité.
          </Text>
        </ScrollView>
      )}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  content: { padding: spacing.lg },
  hero: { alignItems: "center", marginVertical: spacing.xl },
  name: { fontSize: 26, fontWeight: "700", marginTop: spacing.md, letterSpacing: -0.4 },
  email: { fontSize: fontSizes.sm, marginTop: 2 },
  badge: {
    marginTop: spacing.md,
    fontSize: fontSizes.xs,
    fontWeight: "600",
    paddingHorizontal: 12,
    paddingVertical: 6,
    borderRadius: 999,
    overflow: "hidden"
  },
  group: { borderWidth: 1, borderRadius: 20, overflow: "hidden" },
  free: { fontSize: fontSizes.xs, textAlign: "center", marginTop: spacing.xl, lineHeight: 18 }
});
