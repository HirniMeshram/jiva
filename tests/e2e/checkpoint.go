package main

import "time"

func checkpointTest() {
	ControllerIP, Replica1IP, Replica2IP, Replica3IP := "172.18.0.40", "172.18.0.41", "172.18.0.42", "172.18.0.43"
	config := buildConfig(ControllerIP, []string{Replica1IP, Replica2IP, Replica3IP})
	setupTest(config)
	config.verifyRWReplicaCount(3)
	table := config.writeRandomData()

	config.Image = getJivaDebugImageID()
	config.ReplicaEnvs["PANIC_WHILE_SETTING_CHECKPOINT"] = "true"
	config.restartReplicas(Replica3IP)
	verifyRestartCount(config.Replicas[Replica3IP], 1)
	delete(config.ReplicaEnvs, "PANIC_WHILE_SETTING_CHECKPOINT")
	stopContainer(config.Replicas[Replica3IP])

	//To create additional snapshot, 1 more than Replica3
	config.Image = getJivaImageID()
	config.restartReplicas(Replica1IP)
	config.verifyRWReplicaCount(2) // Replica 1 and Replica 2 are in RW

	// Stop Replica 1 and 2 to create Replica 3 as master with 1 less snapshot,
	// with no checkpoint,but same Revision Count
	stopContainer(config.Replicas[Replica2IP])
	stopContainer(config.Replicas[Replica1IP])

	// Replica 1 will be master now
	startContainer(config.Replicas[Replica3IP])
	time.Sleep(5 * time.Second)

	startContainer(config.Replicas[Replica2IP])
	startContainer(config.Replicas[Replica1IP])
	config.verifyRWReplicaCount(3)
	config.verifyRandomData(table)
	scrap(config)
}
