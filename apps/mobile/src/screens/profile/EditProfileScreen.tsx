import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import { useQueryClient } from "@tanstack/react-query";
import { profileApi } from "../../api/endpoints";
import type { MyProfile } from "../../api/types";
import { AppText } from "../../components/AppText";
import { Button } from "../../components/Button";
import { Header } from "../../components/Header";
import { InterestsPicker } from "../../components/profile/InterestsPicker";
import { draftFromProfile, draftToInput, ProfileForm, validateDraft, type DraftErrors } from "../../components/profile/ProfileForm";
import { Screen } from "../../components/Screen";
import { LoadingState } from "../../components/States";
import { showToast } from "../../components/Toast";
import { useInterests, useMyProfile } from "../../hooks/queries";
import { queryKeys } from "../../queryClient";
import { colors, spacing } from "../../theme/tokens";
import { errorMessage } from "../../utils/errors";

function Editor({ data }: { data: MyProfile }) {
  const navigation = useNavigation();
  const client = useQueryClient();
  const catalog = useInterests();
  const [draft, setDraft] = useState(() => draftFromProfile(data.profile));
  const [interestIds, setInterestIds] = useState(() => data.interests.map((i) => i.id));
  const [errors, setErrors] = useState<DraftErrors>({});
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const save = async () => {
    const found = validateDraft(draft);
    setErrors(found);
    if (Object.keys(found).length > 0) return;
    setBusy(true);
    setError(null);
    try {
      await profileApi.save(draftToInput(draft));
      await profileApi.setInterests(interestIds);
      await client.invalidateQueries({ queryKey: queryKeys.myProfile });
      showToast("Profil enregistré.", "success");
      navigation.goBack();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll footer={<Button label="Enregistrer" onPress={save} loading={busy} testID="edit-profile-save" />}>
      <Header title="Modifier mes infos" onBack={() => navigation.goBack()} />
      <View style={styles.body}>
        <ProfileForm draft={draft} onChange={setDraft} errors={errors} />
        <AppText variant="heading">Centres d’intérêt</AppText>
        {catalog.data ? <InterestsPicker catalog={catalog.data} selected={interestIds} onChange={setInterestIds} /> : <LoadingState />}
        {error ? <AppText color={colors.danger}>{error}</AppText> : null}
      </View>
    </Screen>
  );
}

export default function EditProfileScreen() {
  const query = useMyProfile();
  return query.data ? <Editor data={query.data} /> : <LoadingState />;
}

const styles = StyleSheet.create({
  body: { gap: spacing.lg, marginTop: spacing.sm }
});
