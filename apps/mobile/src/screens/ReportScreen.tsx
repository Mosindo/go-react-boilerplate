import React, { useState } from "react";
import { View } from "react-native";
import { useNavigation, useRoute, type RouteProp } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { reportUser } from "../api/platform";
import type { ReportReason } from "../api/types";
import { OptionCard } from "../components/OptionCard";
import { errorMessage } from "../i18n/errors";
import { reportReasons } from "../i18n/fr";
import type { RootStackParamList } from "../navigation/types";
import { showSuccess } from "../shared/feedback";
import { Header, ScreenContainer } from "../shared/layout";
import { Button, Input, Notice, Text, spacing } from "../shared/ui";

export default function ReportScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const { params } = useRoute<RouteProp<RootStackParamList, "Report">>();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async () => {
    if (!reason) {
      setError("Choisissez un motif.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await reportUser(params.userId, reason, details.trim());
      showSuccess("Merci, votre signalement a été transmis.");
      navigation.goBack();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <ScreenContainer keyboardAware scroll testID="report-screen">
      <Header onBack={() => navigation.goBack()} title={`Signaler ${params.name}`} />
      <Text tone="muted">
        Votre signalement est confidentiel : la personne ne saura pas que c&apos;est vous. Vous pouvez aussi la bloquer depuis son profil.
      </Text>
      <View style={{ gap: spacing.sm }}>
        {reportReasons.map((item) => (
          <OptionCard key={item.value} label={item.label} onPress={() => setReason(item.value)} selected={reason === item.value} />
        ))}
      </View>
      <View style={{ gap: spacing.xs }}>
        <Text tone="muted" variant="label" weight="medium">
          Précisions (facultatif)
        </Text>
        <Input accessibilityLabel="Précisions" maxLength={1000} multiline onChangeText={setDetails} value={details} />
      </View>
      {error ? <Notice message={error} tone="danger" /> : null}
      <Button
        fullWidth
        label="Envoyer le signalement"
        loading={busy}
        onPress={() => void submit()}
        size="lg"
        testID="report-submit"
        variant="destructive"
      />
    </ScreenContainer>
  );
}
