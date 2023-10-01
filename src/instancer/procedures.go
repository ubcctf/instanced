package instancer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ubcctf/instanced/src/db"
	"github.com/ubcctf/instanced/src/k8s"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func (in *Instancer) LoadCRDs(ctx context.Context) {
	log := in.log
	// Test CRDs
	log.Debug().Msg("querying CRDs")
	var err error
	in.challengeTmpls, err = in.k8sC.QueryInstancedChallenges(ctx, "challenges")
	if err != nil {
		log.Debug().Err(err).Msg("error retrieving challenge definitions from CRDs")
	}
	for k := range in.challengeTmpls {
		log.Info().Str("challenge", k).Msg("parsed challenge template")
	}
	log.Info().Int("count", len(in.challengeTmpls)).Msg("parsed challenges")
}

func (in *Instancer) DestoryExpiredInstances() {
	log := in.log.With().Str("component", "instanced").Logger()
	instances, err := in.dbC.ReadInstanceRecords()
	if err != nil {
		log.Error().Err(err).Msg("error reading instance records")
		return
	}
	log.Info().Int("count", len(instances)).Msg("instances found")
	for _, i := range instances {
		// Any does not marshall properly
		log.Debug().Int64("id", i.Id).Time("expiry", i.Expiry).Str("challenge", i.Challenge).Msg("instance record found")
		if time.Now().After(i.Expiry) {
			log.Info().Int64("id", i.Id).Str("challenge", i.Challenge).Msg("destroying expired instance")
			err := in.DestroyInstance(i)
			if err != nil {
				log.Error().Err(err).Msg("error destroying instance")
			}
		}
	}
}

func (in *Instancer) DestroyInstance(rec db.InstanceRecord) error {
	log := in.log.With().Str("component", "instanced").Logger()
	/* 	chal, ok := in.challengeObjs[rec.Challenge]
	   	if !ok {
	   		return &ChallengeNotFoundError{rec.Challenge}
	   	} */
	chal, err := in.GetChalObjsFromTemplate(rec.Challenge, rec.UUID)
	if err != nil {
		return err
	}

	for _, o := range chal {
		obj := o.DeepCopy()
		// todo: set proper name
		//obj.SetName(fmt.Sprintf("in-%v-%v", obj.GetName(), rec.Id))
		err := in.k8sC.DeleteObject(obj, "challenges")
		if err != nil {
			log.Warn().Err(err).Str("name", obj.GetName()).Str("kind", obj.GetKind()).Msg("error deleting object")
		}
	}
	err = in.dbC.DeleteInstanceRecord(rec.Id)
	if err != nil {
		log.Warn().Err(err).Msg("error deleting instance record")
	}
	return nil
}

func (in *Instancer) CreateInstance(challenge, team string) (db.InstanceRecord, error) {
	log := in.log.With().Str("component", "instanced").Logger()

	/* chal, ok := in.challengeObjs[challenge]
	if !ok {
		return InstanceRecord{}, &ChallengeNotFoundError{challenge}
	} */
	cuuid := uuid.NewString()[0:8]
	chal, err := in.GetChalObjsFromTemplate(challenge, cuuid)
	if err != nil {
		return db.InstanceRecord{}, err
	}

	ttl, err := time.ParseDuration(in.conf.InstanceTTL)
	if err != nil {
		log.Warn().Err(err).Msg("could not parse instance ttl, defaulting to 10 minutes")
		ttl = 10 * time.Minute
	}

	rec, err := in.dbC.InsertInstanceRecord(ttl, team, challenge, cuuid)
	if err != nil {
		log.Error().Err(err).Msg("could not create instance record")
	} else {
		log.Info().Time("expiry", rec.Expiry).
			Str("challenge", rec.Challenge).
			Int64("id", rec.Id).
			Msg("registered new instance")
	}

	var createErr error
	log.Info().Int("count", len(chal)).Msg("creating objects")
	for _, o := range chal {
		obj := o.DeepCopy()
		//obj.SetName(fmt.Sprintf("in-%v-%v", obj.GetName(), rec.Id))
		var resObj *unstructured.Unstructured
		resObj, createErr = in.k8sC.CreateObject(obj, "challenges")
		log.Debug().Any("object", resObj).Msg("created object")
		if createErr != nil {
			log.Error().Err(createErr).Msg("error creating object")
			break
		}
		log.Info().Str("kind", resObj.GetKind()).Str("name", resObj.GetName()).Msg("created object")
	}
	if createErr != nil {
		// todo: handle errors/cleanup incomplete deploys?
		log.Error().Err(err).Msg("could not create an object")
		log.Info().Msg("instance creation incomplete, manual intervention required")
		return db.InstanceRecord{}, errors.New("instance deployment failed")
	}
	return rec, nil
}

func (in *Instancer) GetTeamChallengeStates(teamID string) ([]db.InstanceRecord, error) {
	instances, err := in.dbC.ReadInstanceRecordsTeam(teamID)
	if err != nil {
		return nil, err
	}
	//for k := range in.challengeObjs {
	for k := range in.challengeTmpls {
		active := false
		for _, v := range instances {
			if v.Challenge == k {
				active = true
				break
			}
		}
		if !active {
			instances = append(instances, db.InstanceRecord{Expiry: time.Unix(0, 0), Challenge: k, TeamID: teamID})
		}
	}
	return instances, nil
}

type ChalInstIdentifier struct {
	ID string
}

func (in *Instancer) GetChalObjsFromTemplate(chalName string, cuuid string) ([]unstructured.Unstructured, error) {
	tmpl, ok := in.challengeTmpls[chalName]
	if !ok {
		return nil, &ChallengeNotFoundError{chalName}
	}
	var objstr bytes.Buffer
	tmpl.Execute(&objstr, ChalInstIdentifier{ID: cuuid})
	chal, err := k8s.UnmarshalManifestFile(objstr.String())
	if err != nil {
		return nil, fmt.Errorf("could not parse challenge: %q : %w", chalName, err)
	}
	return chal, nil
}

/*
func (in *Instancer) ParseTemplates() {
	in.challengeTmpls = make(map[string]*template.Template, len(in.conf.Challenges))
	for k, v := range in.conf.Challenges {
		tmpl, err := template.New("challenge").Parse(v)
		if err != nil {
			in.log.Error().Err(err).Str("challenge", k).Msg("could not parse a challenge template")
			continue
		}
		in.challengeTmpls[k] = tmpl
	}
}
*/
