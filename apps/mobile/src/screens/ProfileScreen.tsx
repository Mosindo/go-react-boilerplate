import React from "react";
import { ScrollView, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useOwnProfile } from "../hooks/queries";
import { GENDER_LABELS } from "../lib/format";
import type { RootStackParamList } from "../navigation/types";
import { ErrorView, LoadingView } from "../shared/feedback";
import { Header, ScreenContainer } from "../shared/layout";
import { Avatar, Card, ListItem, Text, spacing } from "../shared/ui";
import { useTheme } from "../theme";

type Nav = NativeStackNavigationProp<RootStackParamList>;

export default function ProfileScreen() {
  const { colors } = useTheme();
  const navigation = useNavigation<Nav>();
  const query = useOwnProfile();
  const profile = query.data;

  if (query.isLoading) {
    return (
      <ScreenContainer>
        <LoadingView />
      </ScreenContainer>
    );
  }
  if (query.isError || !profile) {
    return (
      <ScreenContainer>
        <ErrorView message="Impossible de charger votre profil." onAction={() => void query.refetch()} />
      </ScreenContainer>
    );
  }

  const chevron = <Ionicons color={colors.textSubtle} name="chevron-forward" size={20} />;
  const icon = (name: keyof typeof Ionicons.glyphMap) => <Ionicons color={colors.primary} name={name} size={22} />;

  return (
    <ScreenContainer>
      <ScrollView contentContainerStyle={styles.scroll} showsVerticalScrollIndicator={false}>
        <Header title="Mon profil" />
        <Card>
          <View style={styles.identity}>
            <Avatar name={profile.firstName} photoId={profile.photos[0]?.id} size="lg" />
            <View style={styles.identityCopy}>
              <Text variant="heading" weight="bold">
                {profile.firstName}, {profile.age}
              </Text>
              <Text tone="muted">
                {GENDER_LABELS[profile.gender]}
                {profile.city ? ` · ${profile.city}` : ""}
              </Text>
              {!profile.isVisible ? (
                <Text tone="subtle" variant="caption">
                  Profil en pause : vous n'apparaissez plus dans la découverte.
                </Text>
              ) : null}
            </View>
          </View>
          {profile.bio ? <Text tone="muted">{profile.bio}</Text> : null}
        </Card>

        <Card padded={false} style={styles.menu}>
          <ListItem
            leading={icon("create-outline")}
            onPress={() => navigation.navigate("EditProfile")}
            testID="menu-edit"
            title="Modifier mon profil"
            trailing={chevron}
          />
          <ListItem
            leading={icon("images-outline")}
            onPress={() => navigation.navigate("Photos")}
            testID="menu-photos"
            title="Mes photos"
            subtitle={`${profile.photos.length}/6`}
            trailing={chevron}
          />
          <ListItem
            leading={icon("options-outline")}
            onPress={() => navigation.navigate("Preferences")}
            testID="menu-preferences"
            title="Préférences de rencontre"
            trailing={chevron}
          />
          <ListItem
            leading={icon("settings-outline")}
            onPress={() => navigation.navigate("Settings")}
            testID="menu-settings"
            title="Confidentialité et compte"
            trailing={chevron}
          />
        </Card>

        <Text style={styles.free} tone="subtle" variant="caption">
          Lueur est entièrement gratuit : pas d'abonnement, pas de likes limités, pas de publicité.
        </Text>
      </ScrollView>
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  scroll: { gap: spacing.lg, paddingBottom: spacing.xxxl },
  identity: { flexDirection: "row", alignItems: "center", gap: spacing.lg },
  identityCopy: { flex: 1, gap: 2 },
  menu: { padding: spacing.xs },
  free: { textAlign: "center" }
});
