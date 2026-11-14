import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { chatApi, matchingApi, safetyApi } from "../../api/endpoints";
import type { Conversation } from "../../api/types";
import { keys } from "../../state/queryClient";
import { spacing } from "../../theme/palette";
import { Button, InlineError, Sheet, Text, useToast } from "../../ui";
import { ReportSheet } from "./ReportSheet";

type Props = {
  visible: boolean;
  onClose: () => void;
  userId: string;
  firstName: string;
  matchId?: string;
  onViewProfile?: () => void;
  /** Called after unmatch / block / report: the member is gone from view. */
  onRemoved: () => void;
};

type Confirm = "unmatch" | "block" | "clear" | null;

export function SafetyMenu({ visible, onClose, userId, firstName, matchId, onViewProfile, onRemoved }: Props) {
  const qc = useQueryClient();
  const toast = useToast();
  const [confirm, setConfirm] = useState<Confirm>(null);
  const [reporting, setReporting] = useState(false);

  const dropConversation = () => {
    if (!matchId) return;
    qc.setQueryData<Conversation[]>(keys.conversations, (l) => l?.filter((c) => c.matchId !== matchId));
    qc.removeQueries({ queryKey: keys.messages(matchId) });
  };

  const action = useMutation({
    mutationFn: async (kind: Exclude<Confirm, null>) => {
      if (kind === "unmatch" && matchId) await matchingApi.unmatch(matchId);
      if (kind === "block") await safetyApi.block(userId);
      if (kind === "clear" && matchId) await chatApi.clear(matchId);
      return kind;
    },
    onSuccess: (kind) => {
      setConfirm(null);
      onClose();
      if (kind === "clear") {
        void qc.invalidateQueries({ queryKey: keys.messages(matchId ?? "") });
        void qc.invalidateQueries({ queryKey: keys.conversations });
        toast({ title: "Conversation effacée de votre côté", tone: "success" });
        return;
      }
      dropConversation();
      void qc.invalidateQueries({ queryKey: keys.blocks });
      toast({ title: kind === "block" ? `${firstName} est bloqué·e` : "Match annulé", tone: "success" });
      onRemoved();
    }
  });

  const confirmCopy: Record<Exclude<Confirm, null>, { title: string; body: string; cta: string }> = {
    unmatch: { title: `Annuler le match avec ${firstName} ?`, body: "La conversation disparaîtra pour vous deux. Vous ne vous recroiserez plus dans Découvrir.", cta: "Annuler le match" },
    block: { title: `Bloquer ${firstName} ?`, body: "Vous ne verrez plus son profil et il ou elle ne pourra plus vous voir ni vous écrire. Vous pourrez débloquer plus tard.", cta: "Bloquer" },
    clear: { title: "Effacer la conversation ?", body: "Les messages seront effacés uniquement de votre côté. Le match est conservé.", cta: "Effacer" }
  };

  return (
    <>
      <Sheet onClose={() => (confirm ? setConfirm(null) : onClose())} title={confirm ? confirmCopy[confirm].title : firstName} visible={visible && !reporting}>
        {confirm ? (
          <View style={styles.list}>
            <Text tone="muted">{confirmCopy[confirm].body}</Text>
            <InlineError message={action.isError ? errorMessage(action.error) : null} />
            <Button label={confirmCopy[confirm].cta} loading={action.isPending} onPress={() => action.mutate(confirm)} variant="danger" />
            <Button label="Retour" onPress={() => setConfirm(null)} variant="ghost" />
          </View>
        ) : (
          <View style={styles.list}>
            {onViewProfile ? <Button icon="person-outline" label="Voir le profil" onPress={onViewProfile} variant="outline" /> : null}
            {matchId ? <Button icon="trash-outline" label="Effacer la conversation (pour moi)" onPress={() => setConfirm("clear")} variant="outline" /> : null}
            {matchId ? <Button icon="heart-dislike-outline" label="Annuler le match" onPress={() => setConfirm("unmatch")} variant="outline" /> : null}
            <Button icon="hand-left-outline" label={`Bloquer ${firstName}`} onPress={() => setConfirm("block")} variant="outline" />
            <Button icon="flag-outline" label={`Signaler ${firstName}`} onPress={() => setReporting(true)} variant="danger" />
          </View>
        )}
      </Sheet>
      <ReportSheet
        firstName={firstName}
        onClose={() => setReporting(false)}
        onDone={() => {
          setReporting(false);
          onClose();
          dropConversation();
          onRemoved();
        }}
        userId={userId}
        visible={reporting}
      />
    </>
  );
}

const styles = StyleSheet.create({
  list: { gap: spacing.xs }
});
