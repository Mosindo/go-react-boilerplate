import { conversation, matchItem } from "../../test-utils";
import { selectNewMatches } from "./newMatches";

it("keeps matches that have no conversation with messages yet", () => {
  const matches = [
    matchItem({ matchId: "1", conversationId: "c1" }),
    matchItem({ matchId: "2", conversationId: "c2" })
  ];
  expect(selectNewMatches(matches, [conversation({ id: "c1" })]).map((m) => m.matchId)).toEqual(["2"]);
});
