import React from "react";
import { View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { AppText, Avatar, Button, Card, Chip, Row, Screen } from "../../components/ui";
import { useMyProfile, useUnreadNotifications } from "../../hooks";
import { spacing } from "../../theme/theme";
import type { MainStackParams } from "../../navigation/types";

export default function MeScreen() {
  const nav = useNavigation<NativeStackNavigationProp<MainStackParams>>();
  const { data: profile } = useMyProfile();
  const unread = useUnreadNotifications();
  if (!profile)
    return (
      <Screen>
        <View />
      </Screen>
    );

  return (
    <Screen scroll>
      <Row>
        <Avatar name={profile.firstName} uri={profile.photos[0]?.url} size={84} />
        <View style={{ flex: 1, gap: 2 }}>
          <AppText variant="title" testID="me-name">
            {profile.firstName}, {profile.age}
          </AppText>
          <AppText variant="muted">{profile.city || "Ville non renseignée"}</AppText>
          {!profile.discoverable ? <AppText variant="muted">Profil en pause</AppText> : null}
        </View>
      </Row>
      {profile.bio ? <AppText>{profile.bio}</AppText> : null}
      <Row style={{ flexWrap: "wrap", gap: spacing.sm }}>
        {profile.interests.map((i) => (
          <Chip key={i.slug} label={i.label} />
        ))}
      </Row>
      <Button
        testID="edit-profile"
        title="Modifier mon profil et mes photos"
        onPress={() => nav.navigate("EditProfile")}
      />
      <Button
        testID="notifications"
        title={unread > 0 ? `Notifications (${unread})` : "Notifications"}
        variant="secondary"
        onPress={() => nav.navigate("Notifications")}
      />
      <Button
        testID="settings"
        title="Compte et confidentialité"
        variant="secondary"
        onPress={() => nav.navigate("Settings")}
      />
      <Card>
        <AppText variant="heading">100 % gratuit</AppText>
        <AppText variant="muted">
          Aucun abonnement, aucun boost payant, aucune limite de likes. Toutes les fonctionnalités sont
          ouvertes à tout le monde.
        </AppText>
      </Card>
    </Screen>
  );
}
