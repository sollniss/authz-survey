package main

import (
	"context"
	"log"

	rts "github.com/ory/keto/proto/ory/keto/relation_tuples/v1alpha2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	wconn, err := grpc.NewClient("127.0.0.1:4467", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("new write conn: " + err.Error())
	}

	ctx := context.Background()
	writer := rts.NewWriteServiceClient(wconn)

	var (
		companyNamespace = "companies"
		CAN_MANAGE       = "MANAGE"
		CAN_VIEW         = "VIEW"
	)

	// reset everything
	_, err = writer.DeleteRelationTuples(ctx, &rts.DeleteRelationTuplesRequest{
		RelationQuery: &rts.RelationQuery{
			Namespace: &companyNamespace,
		},
	})
	if err != nil {
		log.Fatal("delete: " + err.Error())
	}

	// create relations
	var tuples []*rts.RelationTuple
	for _, t := range []struct{ n, s, r, o string }{
		{companyNamespace, "alice", CAN_MANAGE, "company-1"},
		{companyNamespace, "bob", CAN_VIEW, "company-1"},
		{companyNamespace, "carol", CAN_MANAGE, "company-2"},
	} {
		tuples = append(tuples, &rts.RelationTuple{
			Namespace: t.n,
			Subject:   rts.NewSubjectID(t.s),
			Relation:  t.r,
			Object:    t.o,
		})
	}
	// create complex relation (anyone who can manage a company can also view it)
	tuples = append(tuples, &rts.RelationTuple{
		Namespace: companyNamespace,
		Subject: rts.NewSubjectSet(
			companyNamespace,
			"company-1",
			CAN_MANAGE,
		),
		Relation: CAN_VIEW,
		Object:   "company-1",
	})

	_, err = writer.TransactRelationTuples(ctx, &rts.TransactRelationTuplesRequest{
		RelationTupleDeltas: rts.RelationTupleToDeltas(tuples, rts.RelationTupleDelta_ACTION_INSERT),
	})
	if err != nil {
		log.Fatal("create: " + err.Error())
	}

	log.Println("Relation tuples inserted successfully:")

	// get all relations
	rconn, err := grpc.NewClient("127.0.0.1:4466", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("new read conn: " + err.Error())
	}
	reader := rts.NewReadServiceClient(rconn)
	res, err := reader.ListRelationTuples(ctx, &rts.ListRelationTuplesRequest{
		RelationQuery: &rts.RelationQuery{
			Namespace: &companyNamespace,
			//Subject:   rts.NewSubjectID("alice"),
			//Relation:  &CAN_MANAGE,
		},
	})
	if err != nil {
		log.Fatal("read: " + err.Error())
	}
	for _, tuple := range res.RelationTuples {
		log.Printf("%s : %s %s %s\n", tuple.Namespace, tuple.Subject.String(), tuple.Relation, tuple.Object)
	}

	// check relations
	checker := rts.NewCheckServiceClient(rconn)
	for _, r := range []struct{ n, s, r, o string }{
		{companyNamespace, "alice", CAN_MANAGE, "company-1"},
		{companyNamespace, "alice", CAN_VIEW, "company-1"},
		{companyNamespace, "bob", CAN_MANAGE, "company-1"},
		{companyNamespace, "carol", CAN_MANAGE, "company-1"},
		{companyNamespace, "carol", CAN_MANAGE, "company-2"},
		// test
		{companyNamespace, "dave", CAN_MANAGE, "company-1"},
		{companyNamespace, "dave", CAN_MANAGE, "company-2"},
	} {
		checkRes, err := checker.Check(ctx, &rts.CheckRequest{
			Tuple: &rts.RelationTuple{
				Namespace: r.n,
				Subject:   rts.NewSubjectID(r.s),
				Relation:  r.r,
				Object:    r.o,
			},
		})
		if err != nil {
			log.Fatal("check: " + err.Error())
		}
		if checkRes.Allowed {
			log.Printf("%s is allowed to %s %s\n", r.s, r.r, r.o)
		} else {
			log.Printf("%s is not allowed to %s %s\n", r.s, r.r, r.o)
		}
	}
}

func p[T any](v T) *T {
	return &v
}
