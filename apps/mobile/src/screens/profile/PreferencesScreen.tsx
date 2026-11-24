import React, { useState } from "react";
import { useNavigation } from "@react-navigation/native";
import { useQueryClient } from "@tanstack/react-query";
import { profileApi } from "../../api/endpoints";
import type { MyProfile, Preferences } from "../../api/types";
import { AppText } from "../../components/AppText";
import { Button } from "../../components/Button";
import { Header } from "../../components/Header";
import { PreferencesForm } from "../../components/profile/PreferencesForm";
import { Screen } from "../../components/Screen";
import { LoadingState } from "../../components/States";
import { showToast } from "../../components/Toast";
import { useMyProfile } from "../../hooks/queries";
import { queryKeys } from "../../queryClient";
import { colors } from "../../theme/tokens";
import { errorMessage } from "../../utils/errors";

function Editor({ initial }: { initial: Preferences }) {
  const navigation = useNavigation();
  const client = useQueryClient();
  const [prefs, setPrefs] = useState(initial);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const save = async () => {
    setBusy(true);
    setError(null);
    try {
      const saved = await profileApi.setPreferences(prefs);
      client.setQueryData<MyProfile>(queryKeys.myProfile, (d) => (d ? { ...d, preferences: saved } : d));
      await client.resetQueries({ queryKey: queryKeys.discovery });
      showToast("Préférences enregistrées.", "success");
      navigation.goBack();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll footer={<Button label="Enregistrer" onPress={save} loading={busy} testID="preferences-save" />}>
      <Header title="Préférences de rencontre" onBack={() => navigation.goBack()} />
      <PreferencesForm value={prefs} onChange={setPrefs} />
      {error ? <AppText color={colors.danger}>{error}</AppText> : null}
    </Screen>
  );
}

export default function PreferencesScreen() {
  const query = useMyProfile();
  return query.data ? <Editor initial={query.data.preferences} /> : <LoadingState />;
}
