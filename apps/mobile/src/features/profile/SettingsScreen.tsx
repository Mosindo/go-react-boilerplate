import React from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { AppStackParamList } from "../../navigation/types";
import { useSession } from "../../state/session";
import { useTheme, type ThemePreference } from "../../theme/ThemeProvider";
import { Card, Header, ListRow, Screen, SectionTitle, Text, useFeedback } from "../../ui";
import { ChoiceList } from "./ChoiceList";

type Props = NativeStackScreenProps<AppStackParamList, "Settings">;

const themes: { value: ThemePreference; label: string }[] = [
  { value: "system", label: "Comme le système" },
  { value: "light", label: "Clair" },
  { value: "dark", label: "Sombre" }
];

export function SettingsScreen({ navigation }: Props) {
  const { user, signOut } = useSession();
  const { preference, setPreference } = useTheme();
  const { confirm } = useFeedback();

  return (
    <Screen scroll>
      <Header title="Paramètres" onBack={navigation.goBack} />
      <SectionTitle>Compte</SectionTitle>
      <Card>
        <ListRow icon="mail-outline" label="E-mail" value={user?.email} />
        <ListRow icon="ban-outline" label="Personnes bloquées" onPress={() => navigation.navigate("BlockedUsers")} />
      </Card>
      <SectionTitle>Apparence</SectionTitle>
      <ChoiceList options={themes} selected={[preference]} onToggle={setPreference} />
      <SectionTitle>Session</SectionTitle>
      <Card>
        <ListRow
          icon="log-out-outline"
          label="Se déconnecter"
          testID="settings-logout"
          onPress={async () => {
            if (await confirm({ title: "Se déconnecter ?", confirmLabel: "Se déconnecter" })) void signOut();
          }}
        />
        <ListRow
          icon="trash-outline"
          label="Supprimer mon compte"
          destructive
          onPress={() => navigation.navigate("DeleteAccount")}
          testID="settings-delete"
        />
      </Card>
      <Text variant="caption" tone="subtle" align="center" style={{ marginTop: 24 }}>
        Lueur · gratuit, sans publicité ni abonnement.
      </Text>
    </Screen>
  );
}
