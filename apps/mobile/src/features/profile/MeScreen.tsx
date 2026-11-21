import React from "react";
import { View } from "react-native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { RootStackParamList, TabParamList } from "../../navigation/types";
import { makeStyles } from "../../theme";
import { Avatar, Card, ErrorState, ListRow, LoadingState, Screen, Section, Text } from "../../ui";
import { genderLabels } from "../../utils/format";
import { useAuth } from "../auth/AuthProvider";
import { useMyProfile } from "./hooks";

type Props = CompositeScreenProps<BottomTabScreenProps<TabParamList, "Me">, NativeStackScreenProps<RootStackParamList>>;

export function MeScreen({ navigation }: Props) {
  const styles = useStyles();
  const { signOut } = useAuth();
  const query = useMyProfile();
  const me = query.data;
  if (query.isLoading) return <LoadingState />;
  if (!me || !me.profile) return <ErrorState message="Profil introuvable." onRetry={() => void query.refetch()} />;
  const p = me.profile;

  return (
    <Screen scroll>
      <View style={styles.hero}>
        <Avatar uri={me.photos[0]?.thumbUrl} name={p.firstName} size={112} ring />
        <Text variant="title" accessibilityRole="header">
          {p.firstName}, {p.age}
        </Text>
        <Text tone="muted">
          {genderLabels[p.gender]}
          {p.city ? ` · ${p.city}` : ""}
        </Text>
        {!p.isVisible ? (
          <View style={styles.paused}>
            <Text variant="caption" tone="primary">
              Profil en pause : vous n&apos;apparaissez pas dans la découverte.
            </Text>
          </View>
        ) : null}
      </View>
      <Section title="Mon profil">
        <Card>
          <ListRow
            testID="me-edit-profile"
            icon="create-outline"
            label="Modifier mon profil"
            description="Prénom, bio, centres d'intérêt…"
            onPress={() => navigation.navigate("EditProfile")}
          />
          <ListRow
            testID="me-photos"
            icon="images-outline"
            label="Mes photos"
            description={`${me.photos.length}/6 photos`}
            onPress={() => navigation.navigate("EditPhotos")}
          />
          <ListRow
            icon="eye-outline"
            label="Aperçu de mon profil"
            onPress={() => navigation.navigate("ProfileDetail", { userId: me.userId, mode: "view" })}
            divider={false}
          />
        </Card>
      </Section>
      <Section title="Rencontres">
        <Card>
          <ListRow
            testID="me-preferences"
            icon="options-outline"
            label="Préférences de rencontre"
            description="Qui, quel âge, quelle distance"
            onPress={() => navigation.navigate("Preferences")}
          />
          <ListRow
            testID="me-settings"
            icon="shield-checkmark-outline"
            label="Confidentialité et sécurité"
            description="Visibilité, blocages, compte"
            onPress={() => navigation.navigate("Settings")}
            divider={false}
          />
        </Card>
      </Section>
      <Section>
        <Card>
          <ListRow testID="me-logout" icon="log-out-outline" label="Se déconnecter" onPress={() => void signOut()} divider={false} />
        </Card>
      </Section>
      <Text variant="caption" tone="subtle" align="center">
        Lueur est entièrement gratuit. Aucun abonnement, aucune option payante.
      </Text>
    </Screen>
  );
}

const useStyles = makeStyles((t) => ({
  hero: { alignItems: "center", gap: t.spacing.sm, paddingVertical: t.spacing.lg },
  paused: {
    backgroundColor: t.colors.primarySoft,
    borderRadius: t.radii.md,
    paddingHorizontal: t.spacing.md,
    paddingVertical: t.spacing.sm,
    marginTop: t.spacing.sm
  }
}));
